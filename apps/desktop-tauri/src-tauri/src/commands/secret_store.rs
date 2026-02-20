use tauri::command;

use crate::security::keychain;

#[command]
pub fn secret_set(provider: String, profile: String, value: String) -> Result<(), String> {
    let service = format!("com.goyais.secrets.{provider}");
    keychain::set_secret(&service, &profile, &value)
}

#[command]
pub fn secret_get(provider: String, profile: String) -> Result<Option<String>, String> {
    let service = format!("com.goyais.secrets.{provider}");
    keychain::get_secret(&service, &profile)
}
