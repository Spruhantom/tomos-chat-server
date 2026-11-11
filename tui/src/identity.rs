//! Linux device identity backed by the desktop Secret Service (for example GNOME Keyring or KWallet).
//! Private key material is never written to TomChat's JSON configuration.
use anyhow::{anyhow, Context, Result};
use keyring::Entry;
use tomchat_noise::NoiseSession;
use zeroize::Zeroize;

const SERVICE: &str = "uk.co.tomos.tomchat";
const ACCOUNT: &str = "device-static-keypair-v1";

pub struct DeviceIdentity {
    private_key: Vec<u8>,
    pub public_key: Vec<u8>,
}

impl DeviceIdentity {
    pub fn load_or_create() -> Result<Self> {
        let entry = Entry::new(SERVICE, ACCOUNT).context("Could not access the Linux Secret Service. Unlock GNOME Keyring/KWallet and try again.")?;
        match entry.get_password() {
            Ok(mut encoded) => {
                let identity = Self::decode(&encoded);
                encoded.zeroize();
                identity
            },
            Err(keyring::Error::NoEntry) => {
                let (mut private_key, public_key) = NoiseSession::generate_static_keypair()
                    .context("Could not generate the device's X25519 identity")?;
                let mut encoded = String::with_capacity(128);
                hex_append(&mut encoded, &private_key);
                hex_append(&mut encoded, &public_key);
                if let Err(error) = entry.set_password(&encoded) {
                    private_key.zeroize();
                    encoded.zeroize();
                    return Err(error).context("Could not save the device identity to the Linux Secret Service");
                }
                private_key.zeroize();
                let identity = Self::decode(&encoded);
                encoded.zeroize();
                identity
            }
            Err(error) => Err(error).context("Could not read the device identity from the Linux Secret Service"),
        }
    }

    fn decode(encoded: &str) -> Result<Self> {
        if encoded.len() != 128 || !encoded.is_ascii() {
            return Err(anyhow!("Stored TomChat identity has an invalid length; refusing to replace it automatically."));
        }
        let mut private_key = hex_decode(&encoded[..64])?;
        let public_key = hex_decode(&encoded[64..])?;
        if private_key.len() != 32 || public_key.len() != 32 {
            private_key.zeroize();
            return Err(anyhow!("Stored TomChat identity is not a valid X25519 keypair."));
        }
        Ok(Self { private_key, public_key })
    }

    pub fn private_key(&self) -> &[u8] { &self.private_key }
}

impl Drop for DeviceIdentity {
    fn drop(&mut self) { self.private_key.zeroize(); }
}

fn hex_append(output: &mut String, bytes: &[u8]) {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    for byte in bytes {
        output.push(HEX[(byte >> 4) as usize] as char);
        output.push(HEX[(byte & 0x0f) as usize] as char);
    }
}

fn hex_decode(value: &str) -> Result<Vec<u8>> {
    if value.len() % 2 != 0 { return Err(anyhow!("Invalid hexadecimal identity data.")); }
    value.as_bytes().chunks_exact(2).map(|pair| {
        let digit = |byte: u8| match byte {
            b'0'..=b'9' => Some(byte - b'0'),
            b'a'..=b'f' => Some(byte - b'a' + 10),
            b'A'..=b'F' => Some(byte - b'A' + 10),
            _ => None,
        };
        let high = digit(pair[0]).ok_or_else(|| anyhow!("Invalid hexadecimal identity data."))?;
        let low = digit(pair[1]).ok_or_else(|| anyhow!("Invalid hexadecimal identity data."))?;
        Ok((high << 4 | low) as u8)
    }).collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hex_encoding_round_trips_key_bytes() {
        let source = [0x00, 0x01, 0x7f, 0x80, 0xff];
        let mut encoded = String::new();
        hex_append(&mut encoded, &source);
        assert_eq!(hex_decode(&encoded).unwrap(), source);
        encoded.zeroize();
    }

    #[test]
    fn malformed_identity_is_rejected_without_silent_regeneration() {
        assert!(DeviceIdentity::decode("not-a-key").is_err());
    }
}
