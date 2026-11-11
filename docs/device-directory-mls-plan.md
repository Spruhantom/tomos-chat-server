# Device identity and group cryptography plan

Status: design only. Device enrolment, group-state validation and complete end-to-end encryption are not implemented by the current TUI.

## Local device identity
- Generate signing and encryption keys locally using maintained cryptographic libraries.
- Protect private keys with Android Keystore or the operating system's secure credential facilities where available.
- Export a versioned signed contact card containing public identity material and a fingerprint.
- Verify contact-card signatures and require out-of-band fingerprint comparison before marking a contact trusted.
- Support device revocation and clear key-change warnings.

## Pairwise messaging
- Select a maintained, independently reviewed asynchronous pairwise protocol.
- Bind session establishment to both verified device identities and fresh nonces.
- Persist ratchet state atomically in encrypted local storage.
- Reject replayed, malformed, oversized or out-of-order protocol messages safely.
- Never expose plaintext or private key material to transport adapters, logs or diagnostics.

## Group messaging
Evaluate OpenMLS for group key management. Define authenticated membership changes, epoch consistency, device addition/removal, recovery and rollback protection before implementation. Validate state transitions on every participating device.

## Direct peer transport
All protocol messages travel over authenticated direct peer connections. A transport adapter is untrusted and carries only encrypted, authenticated protocol envelopes. Do not introduce a relay, hosted directory, rendezvous component or central message queue.

## Required tests
- Invalid signatures and mismatched fingerprints are rejected.
- Replayed handshakes and protocol messages are rejected.
- Key rotation and device revocation take effect locally and propagate peer-to-peer.
- Concurrent state updates cannot corrupt ratchet or group state.
- Crash/restart restores encrypted local state safely.
- Cross-platform Android/Linux interoperability passes known-answer and adversarial tests.

Production security claims require independent review of the complete implementation, not merely a passing unit-test suite.
