use std::collections::hash_map::DefaultHasher;
use std::fs::{self, OpenOptions};
use std::hash::{Hash, Hasher};
use std::io::{Read, Write};
use std::net::{SocketAddr, TcpStream};
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use tauri::{AppHandle, Manager, Runtime};
use tauri_plugin_dialog::{DialogExt, MessageDialogButtons, MessageDialogKind};
use tauri_plugin_shell::{
    process::{CommandChild, CommandEvent},
    ShellExt,
};

const HUB_PORT: u16 = 8787;
const HUB_HEALTH_TIMEOUT: Duration = Duration::from_secs(10);
const HEALTH_POLL_INTERVAL: Duration = Duration::from_millis(250);
const SIDECAR_LOG_FILE: &str = "sidecar.log";
const HUB_DB_FILE: &str = "hub.sqlite3";
const SIDECAR_PROXY_ENV_KEYS: [&str; 8] = [
    "HTTP_PROXY",
    "HTTPS_PROXY",
    "ALL_PROXY",
    "NO_PROXY",
    "http_proxy",
    "https_proxy",
    "all_proxy",
    "no_proxy",
];
const SIDECAR_TLS_ENV_KEYS: [&str; 3] = ["SSL_CERT_FILE", "REQUESTS_CA_BUNDLE", "CURL_CA_BUNDLE"];

#[derive(Default)]
pub struct SidecarState {
    inner: Arc<Mutex<ManagedSidecars>>,
}

#[derive(Default)]
struct ManagedSidecars {
    hub: Option<CommandChild>,
}

pub fn initialize<R: Runtime + 'static>(app: &AppHandle<R>) -> Result<(), String> {
    let app_data_dir = resolve_app_data_dir(app)?;
    fs::create_dir_all(&app_data_dir).map_err(|error| {
        format!(
            "failed to create app data dir {}: {error}",
            app_data_dir.display()
        )
    })?;

    let log_path = app_data_dir.join(SIDECAR_LOG_FILE);
    log_line(&log_path, "initializing sidecar runtime");

    let app_version = app.package_info().version.to_string();
    let internal_token = generate_internal_token();
    let hub_db_path = app_data_dir.join(HUB_DB_FILE);

    let mut started = ManagedSidecars::default();

    let hub_envs = vec![
        ("PORT".to_string(), HUB_PORT.to_string()),
        (
            "HUB_DB_PATH".to_string(),
            hub_db_path.to_string_lossy().into_owned(),
        ),
        ("GOYAIS_VERSION".to_string(), app_version.clone()),
        ("HUB_INTERNAL_TOKEN".to_string(), internal_token.clone()),
    ];
    let mut hub_envs = hub_envs;
    append_passthrough_envs(&mut hub_envs, &SIDECAR_PROXY_ENV_KEYS);
    append_passthrough_envs(&mut hub_envs, &SIDECAR_TLS_ENV_KEYS);
    let (hub_events, hub_child) = app
        .shell()
        .sidecar("goyais-hub")
        .map_err(|error| format!("failed to resolve hub sidecar: {error}"))?
        .envs(hub_envs)
        .spawn()
        .map_err(|error| format!("failed to spawn hub sidecar: {error}"))?;
    spawn_event_logger("hub", hub_events, log_path.clone());
    log_line(&log_path, "hub sidecar spawned");
    started.hub = Some(hub_child);

    if let Err(error) = wait_for_health(HUB_PORT, HUB_HEALTH_TIMEOUT, &log_path) {
        log_line(
            &log_path,
            &format!("hub health probe failed, shutting down sidecars: {error}"),
        );
        kill_children(&mut started, &log_path);
        return Err(error);
    }
    log_line(&log_path, "hub sidecar healthy");

    let state = app.state::<SidecarState>();
    let mut guard = state
        .inner
        .lock()
        .map_err(|_| "failed to lock sidecar state".to_string())?;
    kill_children(&mut guard, &log_path);
    *guard = started;
    log_line(&log_path, "sidecar runtime initialized (hub only)");
    drop(guard);

    Ok(())
}

pub fn shutdown<R: Runtime>(app: &AppHandle<R>) {
    let log_path = resolve_app_data_dir(app)
        .unwrap_or_else(|_| PathBuf::from("."))
        .join(SIDECAR_LOG_FILE);

    if let Ok(mut guard) = app.state::<SidecarState>().inner.lock() {
        kill_children(&mut guard, &log_path);
    } else {
        log_line(&log_path, "failed to lock sidecar state during shutdown");
    }
}

pub fn show_startup_error<R: Runtime>(app: &AppHandle<R>, message: &str) {
    eprintln!("[startup-error] {message}");
    app.dialog()
        .message(message.to_string())
        .title("Goyais Startup Error")
        .kind(MessageDialogKind::Error)
        .buttons(MessageDialogButtons::Ok)
        .show(|_| {});
}

fn wait_for_health(port: u16, timeout: Duration, log_path: &Path) -> Result<(), String> {
    let deadline = Instant::now() + timeout;
    let mut last_error = "health endpoint not ready".to_string();

    while Instant::now() < deadline {
        match probe_health(port) {
            Ok(true) => return Ok(()),
            Ok(false) => last_error = "health payload missing ok=true".to_string(),
            Err(error) => last_error = error,
        }
        thread::sleep(HEALTH_POLL_INTERVAL);
    }

    let message = format!("health check timeout on 127.0.0.1:{port}: {last_error}");
    log_line(log_path, &message);
    Err(message)
}

fn probe_health(port: u16) -> Result<bool, String> {
    let address: SocketAddr = format!("127.0.0.1:{port}")
        .parse()
        .map_err(|error| format!("invalid health address: {error}"))?;
    let mut stream = TcpStream::connect_timeout(&address, Duration::from_secs(1))
        .map_err(|error| format!("connect error: {error}"))?;
    let _ = stream.set_read_timeout(Some(Duration::from_secs(1)));
    let _ = stream.set_write_timeout(Some(Duration::from_secs(1)));

    stream
        .write_all(b"GET /health HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n")
        .map_err(|error| format!("health request write failed: {error}"))?;

    let mut response = Vec::new();
    stream
        .read_to_end(&mut response)
        .map_err(|error| format!("health response read failed: {error}"))?;

    let text = String::from_utf8_lossy(&response);
    let status_ok = text.starts_with("HTTP/1.1 200") || text.starts_with("HTTP/1.0 200");
    if !status_ok {
        return Ok(false);
    }

    let body = text.split("\r\n\r\n").nth(1).unwrap_or_default();
    Ok(body.contains("\"ok\":true") || body.contains("\"ok\": true"))
}

fn spawn_event_logger(
    service: &'static str,
    mut receiver: tauri::async_runtime::Receiver<CommandEvent>,
    log_path: PathBuf,
) {
    tauri::async_runtime::spawn(async move {
        while let Some(event) = receiver.recv().await {
            match event {
                CommandEvent::Stdout(bytes) => {
                    if let Some(line) = normalized_line(&bytes) {
                        log_line(&log_path, &format!("[{service}] stdout: {line}"));
                    }
                }
                CommandEvent::Stderr(bytes) => {
                    if let Some(line) = normalized_line(&bytes) {
                        log_line(&log_path, &format!("[{service}] stderr: {line}"));
                    }
                }
                CommandEvent::Error(error) => {
                    log_line(&log_path, &format!("[{service}] process error: {error}"));
                }
                CommandEvent::Terminated(result) => {
                    log_line(
                        &log_path,
                        &format!(
                            "[{service}] terminated code={:?} signal={:?}",
                            result.code, result.signal
                        ),
                    );
                }
                _ => {}
            }
        }
    });
}

fn normalized_line(bytes: &[u8]) -> Option<String> {
    let line = String::from_utf8_lossy(bytes).trim().to_string();
    if line.is_empty() {
        None
    } else {
        Some(line)
    }
}

fn append_passthrough_envs(envs: &mut Vec<(String, String)>, keys: &[&str]) {
    for key in keys {
        if envs.iter().any(|(name, _)| name == key) {
            continue;
        }
        if let Ok(value) = std::env::var(key) {
            let trimmed = value.trim();
            if !trimmed.is_empty() {
                envs.push(((*key).to_string(), trimmed.to_string()));
            }
        }
    }
}

fn kill_children(sidecars: &mut ManagedSidecars, log_path: &Path) {
    if let Some(hub) = sidecars.hub.take() {
        if let Err(error) = hub.kill() {
            log_line(log_path, &format!("hub sidecar kill failed: {error}"));
        } else {
            log_line(log_path, "hub sidecar stopped");
        }
    }
}

fn resolve_app_data_dir<R: Runtime>(app: &AppHandle<R>) -> Result<PathBuf, String> {
    app.path()
        .app_data_dir()
        .map_err(|error| format!("failed to resolve app_data_dir: {error}"))
}

fn generate_internal_token() -> String {
    let now_nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_nanos())
        .unwrap_or_default();
    let mut hasher = DefaultHasher::new();
    thread::current().id().hash(&mut hasher);
    std::process::id().hash(&mut hasher);
    let entropy = hasher.finish();
    format!("goyais-internal-{now_nanos:x}-{entropy:x}")
}

fn log_line(path: &Path, message: &str) {
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_secs())
        .unwrap_or_default();
    if let Ok(mut file) = OpenOptions::new().create(true).append(true).open(path) {
        let _ = writeln!(file, "[{timestamp}] {message}");
    }
}
