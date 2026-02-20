use std::sync::Mutex;

#[derive(Default)]
pub struct RuntimeState {
    pub pid: Mutex<Option<u32>>,
}
