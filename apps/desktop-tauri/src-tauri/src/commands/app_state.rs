use std::collections::HashMap;
use std::sync::Mutex;

#[derive(Default)]
pub struct RuntimeState {
    pub service_pid: Mutex<HashMap<String, u32>>,
}
