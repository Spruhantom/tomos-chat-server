package uk.co.tomos.chat

import java.io.IOException
import java.net.InetSocketAddress
import java.net.ServerSocket
import java.net.Socket
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Direct Noise-only transport shared with the Linux Rust peer adapter.
 *
 * The trusted contact card must have been signature-validated and its full
 * signing-key fingerprint independently compared before this API is called.
 * The card's signed X25519 key is pinned by the Noise XX handshake itself.
 * There is deliberately no account server, relay, directory or fallback.
 */
class DirectNoisePeerListener(
    private val bindAddress: InetSocketAddress,
    private val localIdentity: NoiseIdentity,
    private val timeoutMs: Int = 8_000
) : AutoCloseable {
    private val server = ServerSocket()
    private val closed = AtomicBoolean(false)

    @Synchronized
    @Throws(IOException::class)
    fun start() {
        require(timeoutMs in 1..30_000)
        check(!server.isBound) { "Listener has already started." }
        server.reuseAddress = false
        server.bind(bindAddress, 1)
        server.soTimeout = 0 // Keep listening until explicitly stopped; handshake itself is time-bounded.
    }

    @Throws(IOException::class)
    fun accept(trustedPeer: TrustedContact): EncryptedPeerChannel {
        check(server.isBound && !closed.get()) { "Direct listener is not active." }
        require(trustedPeer.noiseStaticPublicKey?.size == 32) {
            "Trusted contact has no signed Noise static key."
        }
        val socket = server.accept()
        socket.tcpNoDelay = true
        socket.soTimeout = timeoutMs
        val connection = AuthenticatedPeerConnection(socket, trustedPeer.fingerprint)
        val session = NoisePeerSession.create(localIdentity, trustedPeer, initiator = false)
        return connection.establishEncryptedChannel(session, initiator = false, handshakeTimeoutMs = timeoutMs)
    }

    val localPort: Int get() = server.localPort

    override fun close() {
        if (closed.compareAndSet(false, true)) {
            try { server.close() } catch (_: Exception) { }
        }
    }
}

object DirectNoisePeerConnector {
    @Throws(IOException::class)
    fun connect(
        address: String,
        port: Int,
        localIdentity: NoiseIdentity,
        trustedPeer: TrustedContact,
        timeoutMs: Int = 8_000
    ): EncryptedPeerChannel {
        require(address.isNotBlank() && address.length <= 253)
        require(port in 1..65535 && timeoutMs in 1..30_000)
        require(trustedPeer.noiseStaticPublicKey?.size == 32) {
            "Trusted contact has no signed Noise static key."
        }
        val socket = Socket()
        try {
            socket.connect(InetSocketAddress(address, port), timeoutMs)
            socket.tcpNoDelay = true
            socket.soTimeout = timeoutMs
            val connection = AuthenticatedPeerConnection(socket, trustedPeer.fingerprint)
            val session = NoisePeerSession.create(localIdentity, trustedPeer, initiator = true)
            return connection.establishEncryptedChannel(session, initiator = true, handshakeTimeoutMs = timeoutMs)
        } catch (error: Exception) {
            try { socket.close() } catch (_: Exception) { }
            if (error is IOException) throw error
            throw IOException("Direct Noise peer connection failed.", error)
        }
    }
}
