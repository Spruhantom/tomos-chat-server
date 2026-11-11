package uk.co.tomos.chat

import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.bouncycastle.crypto.params.X25519PrivateKeyParameters
import java.net.InetSocketAddress
import java.security.SecureRandom
import java.util.UUID
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit

@RunWith(AndroidJUnit4::class)
class DirectNoiseInteropTransportInstrumentedTest {
    private fun identity(): NoiseIdentity {
        val key = X25519PrivateKeyParameters(SecureRandom())
        return NoiseIdentity(key.encoded, key.generatePublicKey().encoded)
    }

    private fun contact(name: String, identity: NoiseIdentity): TrustedContact {
        val signingKey = byteArrayOf(1, 2, 3, 4)
        return TrustedContact(
            displayName = name,
            encodedPublicKey = signingKey,
            fingerprint = ContactCardCodec.fingerprint(signingKey),
            noiseStaticPublicKey = identity.publicKey
        )
    }

    @Test
    fun directNoiseOnlyListenerAndConnectorExchangeEncryptedAppFrames() {
        val alice = identity()
        val bob = identity()
        val aliceKnowsBob = contact("Bob", bob)
        val bobKnowsAlice = contact("Alice", alice)
        val listener = DirectNoisePeerListener(InetSocketAddress("127.0.0.1", 0), bob)
        listener.start()
        val pool = Executors.newSingleThreadExecutor()
        try {
            val accepted = pool.submit<EncryptedPeerChannel> { listener.accept(bobKnowsAlice) }
            val outgoing = DirectNoisePeerConnector.connect(
                "127.0.0.1", listener.localPort, alice, aliceKnowsBob
            )
            val incoming = accepted.get(10, TimeUnit.SECONDS)
            try {
                val sent = EncryptedAppFrame(EncryptedAppFrame.Type.TEXT, UUID.randomUUID(), "cross-platform-ready".toByteArray())
                outgoing.send(sent)
                val received = incoming.receive()
                assertEquals(sent.messageId, received.messageId)
                assertArrayEquals(sent.payload, received.payload)
                val reply = EncryptedAppFrame(EncryptedAppFrame.Type.TEXT, UUID.randomUUID(), "reply".toByteArray())
                incoming.send(reply)
                val replyReceived = outgoing.receive()
                assertEquals(reply.messageId, replyReceived.messageId)
                assertArrayEquals(reply.payload, replyReceived.payload)
            } finally {
                outgoing.close()
                incoming.close()
            }
        } finally {
            listener.close()
            pool.shutdownNow()
            alice.privateKey.fill(0)
            bob.privateKey.fill(0)
        }
    }
}
