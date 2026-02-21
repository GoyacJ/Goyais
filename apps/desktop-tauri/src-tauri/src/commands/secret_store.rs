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

#[command]
pub fn store_token(profile_id: String, token: String) -> Result<(), String> {
    keychain::set_secret("com.goyais.hub.tokens", &profile_id, &token)
}

#[command]
pub fn load_token(profile_id: String) -> Result<Option<String>, String> {
    keychain::get_secret("com.goyais.hub.tokens", &profile_id)
}

#[command]
pub fn delete_token(profile_id: String) -> Result<(), String> {
    keychain::delete_secret("com.goyais.hub.tokens", &profile_id)
}
