use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

use serde::{Deserialize, Serialize};
use tauri::{command, AppHandle, Manager};

const LOCAL_CONFIG_FILE_NAME: &str = "local-process-config.v1.json";

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalProcessConfigV1 {
    pub version: u8,
    pub hub: LocalHubConfig,
    pub runtime: LocalRuntimeConfig,
    pub connections: LocalConnectionConfig,
    pub pending_apply: LocalPendingApplyConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalHubConfig {
    pub port: String,
    pub auth_mode: String,
    pub db_driver: String,
    pub db_path: String,
    pub database_url: String,
    pub worker_base_url: String,
    pub max_concurrent_executions: String,
    pub log_level: String,
    pub advanced_env: HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalRuntimeConfig {
    pub host: String,
    pub port: String,
    pub agent_mode: String,
    pub hub_base_url: String,
    pub require_hub_auth: bool,
    pub workspace_id: String,
    pub workspace_root: String,
    pub sync_server_url: String,
    pub sync_device_id: String,
    pub advanced_env: HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalConnectionConfig {
    pub local_hub_url: String,
    pub default_remote_server_url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LocalPendingApplyConfig {
    pub hub: bool,
    pub runtime: bool,
}

impl Default for LocalProcessConfigV1 {
    fn default() -> Self {
        Self {
            version: 1,
            hub: LocalHubConfig {
                port: "8787".to_string(),
                auth_mode: "local_open".to_string(),
                db_driver: "sqlite".to_string(),
                db_path: "./data/hub.db".to_string(),
                database_url: String::new(),
                worker_base_url: "http://127.0.0.1:8040".to_string(),
                max_concurrent_executions: "5".to_string(),
                log_level: "info".to_string(),
                advanced_env: HashMap::new(),
            },
            runtime: LocalRuntimeConfig {
                host: "127.0.0.1".to_string(),
                port: "8040".to_string(),
                agent_mode: "vanilla".to_string(),
                hub_base_url: "http://127.0.0.1:8787".to_string(),
                require_hub_auth: true,
                workspace_id: "local".to_string(),
                workspace_root: ".".to_string(),
                sync_server_url: "http://127.0.0.1:8140".to_string(),
                sync_device_id: "local-device".to_string(),
                advanced_env: HashMap::new(),
            },
            connections: LocalConnectionConfig {
                local_hub_url: "http://127.0.0.1:8787".to_string(),
                default_remote_server_url: "http://127.0.0.1:8787".to_string(),
            },
            pending_apply: LocalPendingApplyConfig {
                hub: false,
                runtime: false,
            },
        }
    }
}

#[command]
pub fn local_config_read(app: AppHandle) -> Result<LocalProcessConfigV1, String> {
    let path = config_file_path(&app)?;
    if !path.exists() {
        return Ok(LocalProcessConfigV1::default());
    }

    let raw = fs::read_to_string(&path).map_err(|error| error.to_string())?;
    serde_json::from_str::<LocalProcessConfigV1>(&raw).map_err(|error| error.to_string())
}

#[command]
pub fn local_config_write(
    app: AppHandle,
    config: LocalProcessConfigV1,
) -> Result<LocalProcessConfigV1, String> {
    let path = config_file_path(&app)?;
    let content = serde_json::to_string_pretty(&config).map_err(|error| error.to_string())?;
    fs::write(path, content).map_err(|error| error.to_string())?;
    Ok(config)
}

fn config_file_path(app: &AppHandle) -> Result<PathBuf, String> {
    let mut dir = app
        .path()
        .app_config_dir()
        .map_err(|error| error.to_string())?;
    fs::create_dir_all(&dir).map_err(|error| error.to_string())?;
    dir.push(LOCAL_CONFIG_FILE_NAME);
    Ok(dir)
}
