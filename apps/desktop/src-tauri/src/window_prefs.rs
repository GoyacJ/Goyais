const TITLEBAR_DOUBLE_CLICK_ACTION_MINIMIZE: &str = "minimize";
const TITLEBAR_DOUBLE_CLICK_ACTION_ZOOM: &str = "zoom";

#[tauri::command]
pub fn get_macos_titlebar_double_click_action() -> String {
    resolve_macos_titlebar_double_click_action()
}

#[cfg(target_os = "macos")]
fn resolve_macos_titlebar_double_click_action() -> String {
    if let Some(action) = read_string_preference("AppleActionOnDoubleClick")
        .and_then(|value| parse_titlebar_double_click_action(&value))
    {
        return action;
    }

    if let Some(minimize_on_double_click) = read_boolean_preference("AppleMiniaturizeOnDoubleClick") {
        if minimize_on_double_click {
            return TITLEBAR_DOUBLE_CLICK_ACTION_MINIMIZE.to_owned();
        }
    }

    TITLEBAR_DOUBLE_CLICK_ACTION_ZOOM.to_owned()
}

#[cfg(not(target_os = "macos"))]
fn resolve_macos_titlebar_double_click_action() -> String {
    TITLEBAR_DOUBLE_CLICK_ACTION_ZOOM.to_owned()
}

#[cfg(target_os = "macos")]
fn parse_titlebar_double_click_action(value: &str) -> Option<String> {
    let normalized = value.trim().to_ascii_lowercase();
    if normalized.contains("minimi") {
        return Some(TITLEBAR_DOUBLE_CLICK_ACTION_MINIMIZE.to_owned());
    }

    if normalized.contains("maxim") || normalized.contains("zoom") || normalized.contains("fill") {
        return Some(TITLEBAR_DOUBLE_CLICK_ACTION_ZOOM.to_owned());
    }

    None
}

#[cfg(target_os = "macos")]
fn read_string_preference(key: &str) -> Option<String> {
    let value = read_global_preference(key)?;
    if value.is_empty() {
        return None;
    }
    Some(value)
}

#[cfg(target_os = "macos")]
fn read_boolean_preference(key: &str) -> Option<bool> {
    let value = read_global_preference(key)?;
    let normalized = value.trim().to_ascii_lowercase();
    match normalized.as_str() {
        "1" | "true" | "yes" => Some(true),
        "0" | "false" | "no" => Some(false),
        _ => None,
    }
}

#[cfg(target_os = "macos")]
fn read_global_preference(key: &str) -> Option<String> {
    let output = std::process::Command::new("defaults")
        .args(["read", "-g", key])
        .output()
        .ok()?;

    if !output.status.success() {
        return None;
    }

    let value = String::from_utf8_lossy(&output.stdout).trim().to_owned();
    if value.is_empty() {
        return None;
    }
    Some(value)
}
