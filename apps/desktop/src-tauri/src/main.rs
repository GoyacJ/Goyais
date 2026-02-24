#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod sidecar;

use std::io;

use tauri::RunEvent;
use tauri_plugin_autostart::MacosLauncher;

fn main() {
    let app = tauri::Builder::default()
        .manage(sidecar::SidecarState::default())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            None::<Vec<&str>>,
        ))
        .setup(|app| {
            let handle = app.handle().clone();
            if let Err(error) = sidecar::initialize(&handle) {
                sidecar::show_startup_error(&handle, &error);
                return Err(io::Error::other(error).into());
            }
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building tauri application");

    app.run(|app_handle, event| {
        if matches!(event, RunEvent::ExitRequested { .. } | RunEvent::Exit) {
            sidecar::shutdown(app_handle);
        }
    });
}
