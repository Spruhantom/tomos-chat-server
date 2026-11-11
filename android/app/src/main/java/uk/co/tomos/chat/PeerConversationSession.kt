package uk.co.tomos.chat

import java.io.IOException
import java.nio.charset.CodingErrorAction
import java.util.UUID
import java.util.concurrent.atomic.AtomicReference

enum class PeerConnectionStatus { ENCRYPTING, CONNECTED, DISCONNECTED, FAILED }

/**
 * A conversation-scoped adapter over an already authenticated Noise channel.
 * All application frames go through EncryptedPeerChannel; callers should run
 * blocking receive() on Dispatchers.IO and publish returned messages to UI.
 */
class PeerConversationSession(
    private val peerFingerprint: String,
    private val channel: EncryptedPeerChannel,
    private val history: EncryptedMessageStore,
    private val clock: () -> Long = System::currentTimeMillis
) : AutoCloseable {
    private val statusRef = AtomicReference(PeerConnectionStatus.CONNECTED)

    init {
        require(peerFingerprint.isNotBlank() && peerFingerprint.length <= 128)
        check(channel.isOpen) { "Conversation requires an established encrypted channel." }
    }

    val status: PeerConnectionStatus
        get() = if (statusRef.get() == PeerConnectionStatus.CONNECTED && !channel.isOpen) {
            statusRef.compareAndSet(PeerConnectionStatus.CONNECTED, PeerConnectionStatus.DISCONNECTED)
            statusRef.get()
        } else statusRef.get()

    fun sendText(text: String): StoredChatMessage {
        check(status == PeerConnectionStatus.CONNECTED) {
            "Messages cannot be sent until the encrypted peer connection is established."
        }
        require(text.isNotBlank()) { "Message cannot be empty." }
        require(text.toByteArray(Charsets.UTF_8).size <= AppFrameCodec.MAX_PAYLOAD_BYTES)
        val message = StoredChatMessage(
            id = UUID.randomUUID(),
            peerFingerprint = peerFingerprint,
            direction = StoredMessageDirection.OUTGOING,
            status = StoredMessageStatus.QUEUED,
            createdAtEpochMillis = clock(),
            body = text
        )
        history.append(message)
        return try {
            channel.send(EncryptedAppFrame(EncryptedAppFrame.Type.TEXT, message.id, text.toByteArray(Charsets.UTF_8)))
            // A successful socket write is not delivery. Keep QUEUED until the
            // remote peer persists the message and returns its encrypted ACK.
            message
        } catch (error: Exception) {
            statusRef.set(PeerConnectionStatus.FAILED)
            try { history.updateStatus(message.id, StoredMessageStatus.FAILED) } catch (_: IOException) { }
            throw error
        }
    }

    @Throws(IOException::class)
    fun receiveText(onAcknowledged: (UUID) -> Unit = {}): StoredChatMessage {
        check(status == PeerConnectionStatus.CONNECTED) { "Encrypted peer connection is not active." }
        return try {
            while (true) {
                val frame = channel.receive()
                if (frame.type == EncryptedAppFrame.Type.ACKNOWLEDGEMENT) {
                    require(frame.payload.isEmpty()) { "Acknowledgement payload must be empty." }
                    history.updateStatus(frame.messageId, StoredMessageStatus.SENT)
                    onAcknowledged(frame.messageId)
                    continue
                }
                require(frame.type == EncryptedAppFrame.Type.TEXT) { "Unexpected non-text frame in a text conversation." }
            val decoder = Charsets.UTF_8.newDecoder()
                .onMalformedInput(CodingErrorAction.REPORT)
                .onUnmappableCharacter(CodingErrorAction.REPORT)
            val body = decoder.decode(java.nio.ByteBuffer.wrap(frame.payload)).toString()
            require(body.isNotBlank()) { "Received empty text message." }
            val message = StoredChatMessage(
                id = frame.messageId,
                peerFingerprint = peerFingerprint,
                direction = StoredMessageDirection.INCOMING,
                status = StoredMessageStatus.RECEIVED,
                createdAtEpochMillis = clock(),
                body = body
            )
            history.append(message)
            channel.send(EncryptedAppFrame(EncryptedAppFrame.Type.ACKNOWLEDGEMENT, message.id, byteArrayOf()))
            return message
            }
            error("Unreachable receive state.")
        } catch (error: Exception) {
            statusRef.set(PeerConnectionStatus.FAILED)
            if (error is IOException) throw error
            throw IOException("Encrypted conversation frame was rejected.", error)
        }
    }

    /** Re-send locally queued messages after a new authenticated connection. */
    fun retryQueuedMessages() {
        check(status == PeerConnectionStatus.CONNECTED)
        history.load(peerFingerprint)
            .filter { it.direction == StoredMessageDirection.OUTGOING && it.status == StoredMessageStatus.QUEUED }
            .forEach { message ->
                channel.send(EncryptedAppFrame(EncryptedAppFrame.Type.TEXT, message.id, message.body.toByteArray(Charsets.UTF_8)))
            }
    }

    override fun close() {
        statusRef.set(PeerConnectionStatus.DISCONNECTED)
        channel.close()
    }
}
