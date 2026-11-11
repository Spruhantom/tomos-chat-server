use crate::config::{Config, Peer};
use anyhow::Result;

pub struct App {
    pub config: Config,
    pub should_quit: bool,
    pub input_mode: bool,
    pub input: String,
    pub status: String,
    pub selected_peer: usize,
    pub active_panel: usize,
    pub show_help: bool,
    pub messages: Vec<String>,
}

impl App {
    pub fn load() -> Result<Self> {
        Ok(Self {
            config: Config::load()?,
            should_quit: false,
            input_mode: false,
            input: String::new(),
            status: "Offline · direct peer transport not implemented".into(),
            selected_peer: 0,
            active_panel: 0,
            show_help: false,
            messages: vec![
                "Welcome to TomChat TUI.".into(),
                "Decentralised peer-to-peer messaging.".into(),
                "This build is a UI foundation; no messages are sent.".into(),
            ],
        })
    }

    pub fn save(&self) -> Result<()> { self.config.save() }
    pub fn next_panel(&mut self) { self.active_panel = (self.active_panel + 1) % 3; }
    pub fn previous_item(&mut self) {
        if !self.config.peers.is_empty() { self.selected_peer = self.selected_peer.saturating_sub(1); }
    }
    pub fn next_item(&mut self) {
        if !self.config.peers.is_empty() { self.selected_peer = (self.selected_peer + 1).min(self.config.peers.len() - 1); }
    }
    pub fn submit_input(&mut self) {
        let line = self.input.trim().to_string();
        self.input.clear();
        self.input_mode = false;
        if line.is_empty() { return; }
        if let Some(command) = line.strip_prefix('/') { self.command(command); }
        else { self.messages.push("Not sent: direct transport and end-to-end encryption are not implemented yet.".into()); }
    }
    fn command(&mut self, command: &str) {
        let mut parts = command.split_whitespace();
        match parts.next().unwrap_or("") {
            "help" => self.show_help = true,
            "quit" | "exit" => self.should_quit = true,
            "peers" => self.active_panel = 1,
            "chats" => self.active_panel = 0,
            "settings" => self.active_panel = 2,
            "identity" => match crate::identity::DeviceIdentity::load_or_create() {
                Ok(identity) => {
                    let public_key = identity.public_key.iter().map(|byte| format!("{byte:02x}")).collect::<String>();
                    self.status = format!("Device identity ready · X25519 public key: {public_key}");
                }
                Err(error) => self.status = format!("Identity unavailable: {error:#}"),
            },
            "add-peer" => {
                let name = parts.next();
                let fingerprint = parts.next();
                let address = parts.next();
                match (name, fingerprint, address) {
                    (Some(name), Some(fingerprint), Some(address)) => {
                        if fingerprint.len() != 64 || !fingerprint.chars().all(|c| c.is_ascii_hexdigit()) {
                            self.status = "Peer fingerprint must be 64 hexadecimal characters.".into();
                        } else {
                            self.config.peers.push(Peer { name: name.to_string(), fingerprint: fingerprint.to_ascii_lowercase(), address: address.to_string() });
                            self.status = format!("Saved local peer hint for {name}; verify the fingerprint out of band before trusting it.");
                        }
                    }
                    _ => self.status = "Usage: /add-peer <name> <sha256-fingerprint> <host:port>".into(),
                }
            }
            "listen" | "connect" => self.status = "Direct networking is planned but not yet implemented.".into(),
            _ => self.status = format!("Unknown command: /{command}. Press F1 for help."),
        }
    }
}
