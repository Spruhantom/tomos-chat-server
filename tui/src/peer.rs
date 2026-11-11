//! Direct TCP peer channel using the shared Noise XX core.
//! No relay, rendezvous or intermediary connection is attempted.

use std::io::{self, Read, Write};
use std::net::{SocketAddr, TcpListener, TcpStream};
use tomchat_noise::{protocol::AppFrame, NoiseSession, NoiseSessionError};

const MAX_HANDSHAKE: usize = 65_535;
const MAX_CIPHERTEXT: usize = 65_535;
const MAX_PLAINTEXT: usize = MAX_CIPHERTEXT - 16;

#[derive(Debug)]
pub enum PeerError { Io(io::Error), Noise(NoiseSessionError), InvalidFrame }
impl From<io::Error> for PeerError { fn from(value: io::Error) -> Self { Self::Io(value) } }
impl From<NoiseSessionError> for PeerError { fn from(value: NoiseSessionError) -> Self { Self::Noise(value) } }
impl std::fmt::Display for PeerError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Io(e) => write!(f, "Direct peer I/O failed: {e}"),
            Self::Noise(e) => write!(f, "Peer authentication/encryption failed: {e}"),
            Self::InvalidFrame => f.write_str("Peer sent an invalid or oversized frame."),
        }
    }
}
impl std::error::Error for PeerError {}

/// Authenticated direct TCP channel. The caller must obtain the remote static
/// key from a signed, fingerprint-verified contact card before connecting.
pub struct PeerChannel { stream: TcpStream, noise: NoiseSession }

impl PeerChannel {
    pub fn connect(address: SocketAddr, local_private: &[u8], peer_static: &[u8]) -> Result<Self, PeerError> {
        let stream = TcpStream::connect(address)?;
        Self::initiator(stream, local_private, peer_static)
    }

    pub fn accept(listener: &TcpListener, local_private: &[u8], peer_static: &[u8]) -> Result<Self, PeerError> {
        let (stream, _) = listener.accept()?;
        Self::responder(stream, local_private, peer_static)
    }

    fn initiator(mut stream: TcpStream, local_private: &[u8], peer_static: &[u8]) -> Result<Self, PeerError> {
        stream.set_nodelay(true)?;
        let mut noise = NoiseSession::initiator(local_private, peer_static)?;
        let first = noise.write_handshake(&[])?;
        write_frame(&mut stream, &first, MAX_HANDSHAKE)?;
        let second = read_frame(&mut stream, MAX_HANDSHAKE)?;
        noise.read_handshake(&second)?;
        let third = noise.write_handshake(&[])?;
        write_frame(&mut stream, &third, MAX_HANDSHAKE)?;
        noise.finish_handshake()?;
        Ok(Self { stream, noise })
    }

    fn responder(mut stream: TcpStream, local_private: &[u8], peer_static: &[u8]) -> Result<Self, PeerError> {
        stream.set_nodelay(true)?;
        let mut noise = NoiseSession::responder(local_private, peer_static)?;
        let first = read_frame(&mut stream, MAX_HANDSHAKE)?;
        noise.read_handshake(&first)?;
        let second = noise.write_handshake(&[])?;
        write_frame(&mut stream, &second, MAX_HANDSHAKE)?;
        let third = read_frame(&mut stream, MAX_HANDSHAKE)?;
        noise.read_handshake(&third)?;
        noise.finish_handshake()?;
        Ok(Self { stream, noise })
    }

    pub fn send(&mut self, plaintext: &[u8]) -> Result<(), PeerError> {
        if plaintext.len() > MAX_PLAINTEXT { return Err(PeerError::InvalidFrame); }
        let ciphertext = self.noise.encrypt(plaintext)?;
        write_frame(&mut self.stream, &ciphertext, MAX_CIPHERTEXT)
    }

    pub fn receive(&mut self) -> Result<Vec<u8>, PeerError> {
        let ciphertext = read_frame(&mut self.stream, MAX_CIPHERTEXT)?;
        Ok(self.noise.decrypt(&ciphertext)?)
    }

    pub fn peer_addr(&self) -> Result<SocketAddr, PeerError> { Ok(self.stream.peer_addr()?) }
}

fn write_frame(stream: &mut TcpStream, bytes: &[u8], limit: usize) -> Result<(), PeerError> {
    if bytes.is_empty() || bytes.len() > limit { return Err(PeerError::InvalidFrame); }
    let length = u32::try_from(bytes.len()).map_err(|_| PeerError::InvalidFrame)?;
    stream.write_all(&length.to_be_bytes())?;
    stream.write_all(bytes)?;
    stream.flush()?;
    Ok(())
}

fn read_frame(stream: &mut TcpStream, limit: usize) -> Result<Vec<u8>, PeerError> {
    let mut header = [0u8; 4];
    stream.read_exact(&mut header)?;
    let length = u32::from_be_bytes(header) as usize;
    if length == 0 || length > limit { return Err(PeerError::InvalidFrame); }
    let mut bytes = vec![0; length];
    stream.read_exact(&mut bytes)?;
    Ok(bytes)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::thread;
    use std::time::Duration;

    #[test]
    fn direct_loopback_noise_xx_exchanges_encrypted_data_both_ways() {
        let (alice_private, alice_public) = NoiseSession::generate_static_keypair().unwrap();
        let (bob_private, bob_public) = NoiseSession::generate_static_keypair().unwrap();
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let address = listener.local_addr().unwrap();
        let bob = thread::spawn(move || {
            let mut channel = PeerChannel::accept(&listener, &bob_private, &alice_public).unwrap();
            assert_eq!(channel.receive().unwrap(), b"hello from Linux client");
            channel.send(b"reply from peer").unwrap();
        });
        let mut alice = PeerChannel::connect(address, &alice_private, &bob_public).unwrap();
        alice.stream.set_read_timeout(Some(Duration::from_secs(3))).unwrap();
        alice.send(b"hello from Linux client").unwrap();
        assert_eq!(alice.receive().unwrap(), b"reply from peer");
        bob.join().unwrap();
    }
}
