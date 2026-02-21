use std::process::Command;

use tauri::command;

#[command]
pub fn git_current_branch(workspace_path: String) -> Result<Option<String>, String> {
    let output = Command::new("git")
        .arg("-C")
        .arg(workspace_path)
        .arg("rev-parse")
        .arg("--abbrev-ref")
        .arg("HEAD")
        .output()
        .map_err(|error| error.to_string())?;

    if !output.status.success() {
        return Ok(None);
    }

    let branch = String::from_utf8_lossy(&output.stdout).trim().to_string();
    if branch.is_empty() || branch == "HEAD" {
        return Ok(None);
    }

    Ok(Some(branch))
}
