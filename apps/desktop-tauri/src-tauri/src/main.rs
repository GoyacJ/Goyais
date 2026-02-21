mod commands {
    pub mod app_state;
    pub mod git;
    pub mod runtime_process;
    pub mod secret_store;
}
mod security {
    pub mod keychain;
}

use commands::app_state::RuntimeState;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(RuntimeState::default())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            commands::runtime_process::runtime_start,
            commands::runtime_process::runtime_status,
            commands::secret_store::secret_get,
            commands::secret_store::secret_set,
            commands::secret_store::store_token,
            commands::secret_store::load_token,
            commands::secret_store::delete_token,
            commands::git::git_current_branch,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn main() {
    run();
}
