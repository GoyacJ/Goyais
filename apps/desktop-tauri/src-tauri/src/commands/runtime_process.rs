use std::process::{Command, Stdio};

use tauri::command;
use tauri::State;

use super::app_state::RuntimeState;

#[command]
pub fn runtime_start(state: State<'_, RuntimeState>, command: String, cwd: String) -> Result<u32, String> {
    let child = Command::new("sh")
        .arg("-lc")
        .arg(command)
        .current_dir(cwd)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .map_err(|error| error.to_string())?;

    let pid = child.id();
    let mut lock = state.pid.lock().map_err(|_| "runtime state poisoned".to_string())?;
    *lock = Some(pid);
    Ok(pid)
}

#[command]
pub fn runtime_status(state: State<'_, RuntimeState>) -> Result<Option<u32>, String> {
    let lock = state.pid.lock().map_err(|_| "runtime state poisoned".to_string())?;
    Ok(*lock)
}
