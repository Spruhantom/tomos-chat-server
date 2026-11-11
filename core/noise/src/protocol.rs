//! Versioned application envelope shared with Android's AppFrameCodec.
use std::convert::TryInto;

pub const MAX_PAYLOAD_BYTES: usize = 16 * 1024;
pub const HEADER_BYTES: usize = 22;
pub const MAX_FRAME_BYTES: usize = HEADER_BYTES + MAX_PAYLOAD_BYTES;

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
#[repr(u8)]
pub enum FrameType {
    Text = 1,
    GroupControl = 2,
    VoiceSignal = 3,
    GroupVoiceControl = 4,
    Acknowledgement = 5,
}

impl TryFrom<u8> for FrameType {
    type Error = ProtocolError;
    fn try_from(value: u8) -> Result<Self, Self::Error> {
        match value {
            1 => Ok(Self::Text),
            2 => Ok(Self::GroupControl),
            3 => Ok(Self::VoiceSignal),
            4 => Ok(Self::GroupVoiceControl),
            5 => Ok(Self::Acknowledgement),
            _ => Err(ProtocolError::InvalidType),
        }
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct AppFrame {
    pub kind: FrameType,
    pub message_id: [u8; 16],
    pub payload: Vec<u8>,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum ProtocolError { InvalidLength, InvalidVersion, InvalidType, InvalidPayload }

impl std::fmt::Display for ProtocolError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(match self {
            Self::InvalidLength => "Invalid application frame length.",
            Self::InvalidVersion => "Unsupported application protocol version.",
            Self::InvalidType => "Unsupported application frame type.",
            Self::InvalidPayload => "Invalid application payload length.",
        })
    }
}
impl std::error::Error for ProtocolError {}

impl AppFrame {
    pub fn encode(&self) -> Result<Vec<u8>, ProtocolError> {
        if self.payload.len() > MAX_PAYLOAD_BYTES { return Err(ProtocolError::InvalidPayload); }
        if self.kind == FrameType::Acknowledgement && !self.payload.is_empty() {
            return Err(ProtocolError::InvalidPayload);
        }
        let mut out = Vec::with_capacity(HEADER_BYTES + self.payload.len());
        out.push(1);
        out.push(self.kind as u8);
        out.extend_from_slice(&self.message_id);
        out.extend_from_slice(&(self.payload.len() as u32).to_be_bytes());
        out.extend_from_slice(&self.payload);
        Ok(out)
    }

    pub fn decode(bytes: &[u8]) -> Result<Self, ProtocolError> {
        if !(HEADER_BYTES..=MAX_FRAME_BYTES).contains(&bytes.len()) { return Err(ProtocolError::InvalidLength); }
        if bytes[0] != 1 { return Err(ProtocolError::InvalidVersion); }
        let kind = FrameType::try_from(bytes[1])?;
        let message_id: [u8; 16] = bytes[2..18].try_into().map_err(|_| ProtocolError::InvalidLength)?;
        let length = u32::from_be_bytes(bytes[18..22].try_into().map_err(|_| ProtocolError::InvalidLength)?) as usize;
        if length > MAX_PAYLOAD_BYTES || length != bytes.len() - HEADER_BYTES { return Err(ProtocolError::InvalidPayload); }
        let payload = bytes[HEADER_BYTES..].to_vec();
        if kind == FrameType::Acknowledgement && !payload.is_empty() { return Err(ProtocolError::InvalidPayload); }
        Ok(Self { kind, message_id, payload })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn android_compatible_text_frame_golden_vector() {
        let frame = AppFrame {
            kind: FrameType::Text,
            message_id: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
            payload: b"hi".to_vec(),
        };
        let expected = vec![
            1, 1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
            0, 0, 0, 2, b'h', b'i'
        ];
        assert_eq!(frame.encode().unwrap(), expected);
        assert_eq!(AppFrame::decode(&expected).unwrap(), frame);
    }

    #[test]
    fn acknowledgement_has_no_payload() {
        let frame = AppFrame { kind: FrameType::Acknowledgement, message_id: [7; 16], payload: vec![] };
        assert_eq!(AppFrame::decode(&frame.encode().unwrap()).unwrap(), frame);
        assert_eq!(AppFrame { payload: vec![1], ..frame }.encode(), Err(ProtocolError::InvalidPayload));
    }

    #[test]
    fn rejects_trailing_bytes_and_oversized_payloads() {
        let mut bytes = AppFrame { kind: FrameType::Text, message_id: [0; 16], payload: vec![] }.encode().unwrap();
        bytes.push(0);
        assert_eq!(AppFrame::decode(&bytes), Err(ProtocolError::InvalidPayload));
        assert_eq!(AppFrame { kind: FrameType::Text, message_id: [0; 16], payload: vec![0; MAX_PAYLOAD_BYTES + 1] }.encode(), Err(ProtocolError::InvalidPayload));
    }
}
