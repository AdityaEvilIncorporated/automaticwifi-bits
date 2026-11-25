package io.github.autowifibits.core

import java.net.Socket
import java.security.KeyStore
import java.security.MessageDigest
import java.security.cert.CertificateException
import java.security.cert.X509Certificate
import javax.net.ssl.SSLContext
import javax.net.ssl.SSLEngine
import javax.net.ssl.SSLSocket
import javax.net.ssl.SSLSocketFactory
import javax.net.ssl.TrustManagerFactory
import javax.net.ssl.X509ExtendedTrustManager
import javax.net.ssl.X509TrustManager

/**
 * Portal TLS (PROTOCOL.md section 6): the platform's default trust manager first and, only when
 * that throws and [pin] is set, a chain whose leaf SPKI SHA-256 equals [pin]. Every connection
 * must be to [host] (the factory cannot be reused for another host). HttpsURLConnection still
 * runs its normal hostname verification on top. Used only for the portal Http; the probe uses
 * the default factory.
 */
object PortalTls {
    private val PIN = Regex("^[0-9a-f]{64}$")

    /** Lowercase hex SHA-256 of the certificate's DER SubjectPublicKeyInfo (same as Go's SPKISHA256). */
    fun spkiSha256(cert: X509Certificate): String =
        MessageDigest.getInstance("SHA-256").digest(cert.publicKey.encoded).joinToString("") { "%02x".format(it) }

    /** [trustStore] null = platform default (tests pass a KeyStore with the mock CA). */
    fun socketFactory(host: String, pin: String, trustStore: KeyStore? = null): SSLSocketFactory {
        require(host.isNotEmpty()) { "empty host" }
        require(pin.isEmpty() || PIN.matches(pin)) { "pinned_cert_sha256 must be 64 lowercase hex characters" }
        val tmf = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm())
        tmf.init(trustStore)
        val base = tmf.trustManagers.filterIsInstance<X509TrustManager>().first()
        val ctx = SSLContext.getInstance("TLS")
        ctx.init(null, arrayOf(PinningTrustManager(base, host, pin)), null)
        return ctx.socketFactory
    }

    internal class PinningTrustManager(
        private val base: X509TrustManager,
        private val host: String,
        private val pin: String,
    ) : X509ExtendedTrustManager() {

        private fun verify(chain: Array<X509Certificate>, peerHost: String?, system: () -> Unit) {
            if (peerHost == null || !peerHost.equals(host, ignoreCase = true)) {
                throw CertificateException("portal TLS config for $host used for $peerHost")
            }
            if (chain.isEmpty()) throw CertificateException("server sent no certificate")
            try {
                system()
            } catch (e: CertificateException) {
                if (pin.isNotEmpty() &&
                    MessageDigest.isEqual(spkiSha256(chain[0]).toByteArray(), pin.toByteArray())
                ) return
                throw e
            }
        }

        override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String, socket: Socket?) {
            val peer = (socket as? SSLSocket)?.handshakeSession?.peerHost
            verify(chain, peer) {
                if (base is X509ExtendedTrustManager) base.checkServerTrusted(chain, authType, socket)
                else base.checkServerTrusted(chain, authType)
            }
        }

        override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String, engine: SSLEngine?) {
            verify(chain, engine?.peerHost) {
                if (base is X509ExtendedTrustManager) base.checkServerTrusted(chain, authType, engine)
                else base.checkServerTrusted(chain, authType)
            }
        }

        // Without a socket the peer host is unknown: refuse rather than skip the host binding.
        override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String) =
            throw CertificateException("portal TLS needs the connection to check the host")

        override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String) =
            throw CertificateException("client certificates are not accepted")

        override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String, socket: Socket?) =
            throw CertificateException("client certificates are not accepted")

        override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String, engine: SSLEngine?) =
            throw CertificateException("client certificates are not accepted")

        override fun getAcceptedIssuers(): Array<X509Certificate> = base.acceptedIssuers
    }
}
