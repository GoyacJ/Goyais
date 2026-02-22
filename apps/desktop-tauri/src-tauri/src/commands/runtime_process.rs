use std::collections::HashMap;
use std::process::{Command, Stdio};

use tauri::command;
use tauri::State;

use super::app_state::RuntimeState;

#[command]
pub fn service_start(
    state: State<'_, RuntimeState>,
    service: String,
    command: String,
    cwd: String,
    env: Option<HashMap<String, String>>,
) -> Result<u32, String> {
    let normalized_service = normalize_service_name(&service)?;
    let child = Command::new("sh")
        .arg("-lc")
        .arg(command)
        .current_dir(cwd)
        .envs(env.unwrap_or_default())
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .map_err(|error| error.to_string())?;

    let pid = child.id();
    let mut lock = state
        .service_pid
        .lock()
        .map_err(|_| "runtime state poisoned".to_string())?;
    lock.insert(normalized_service, pid);
    Ok(pid)
}

#[command]
pub fn service_status(
    state: State<'_, RuntimeState>,
    service: String,
) -> Result<Option<u32>, String> {
    let normalized_service = normalize_service_name(&service)?;
    let lock = state
        .service_pid
        .lock()
        .map_err(|_| "runtime state poisoned".to_string())?;
    Ok(lock.get(&normalized_service).copied())
}

#[command]
pub fn service_stop(state: State<'_, RuntimeState>, service: String) -> Result<(), String> {
    let normalized_service = normalize_service_name(&service)?;
    let mut lock = state
        .service_pid
        .lock()
        .map_err(|_| "runtime state poisoned".to_string())?;

    let Some(pid) = lock.remove(&normalized_service) else {
        return Ok(());
    };

    kill_process(pid)?;
    Ok(())
}

// Backward-compatible command names kept for existing desktop call paths.
#[command]
pub fn runtime_start(
    state: State<'_, RuntimeState>,
    command: String,
    cwd: String,
) -> Result<u32, String> {
    service_start(state, "hub".to_string(), command, cwd, None)
}

#[command]
pub fn runtime_status(state: State<'_, RuntimeState>) -> Result<Option<u32>, String> {
    service_status(state, "hub".to_string())
}

fn normalize_service_name(value: &str) -> Result<String, String> {
    let normalized = value.trim().to_lowercase();
    if normalized == "hub" || normalized == "runtime" {
        return Ok(normalized);
    }
    Err("service must be 'hub' or 'runtime'".to_string())
}

fn kill_process(pid: u32) -> Result<(), String> {
    #[cfg(target_family = "windows")]
    {
        let status = Command::new("taskkill")
            .args(["/PID", &pid.to_string(), "/T", "/F"])
            .status()
            .map_err(|error| error.to_string())?;
        if !status.success() {
            return Err(format!("failed to stop process {}", pid));
        }
    }

    #[cfg(not(target_family = "windows"))]
    {
        let status = Command::new("kill")
            .args(["-TERM", &pid.to_string()])
            .status()
            .map_err(|error| error.to_string())?;
        if !status.success() {
            return Err(format!("failed to stop process {}", pid));
        }
    }

    Ok(())
}
