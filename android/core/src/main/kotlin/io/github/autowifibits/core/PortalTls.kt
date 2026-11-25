package io.github.autowifibits.core

import java.security.KeyStore
import java.security.cert.X509Certificate
import javax.net.ssl.SSLSocketFactory

/**
 * Portal TLS (PROTOCOL.md section 6), HARD task T24 (supervisor): a trust manager that runs
 * the platform's default X509TrustManager with hostname check for [host] first and, only when
 * that throws and [pin] is set, accepts a chain whose leaf SPKI SHA-256 equals [pin].
 * Used only for the portal Http; the probe uses the default factory.
 */
object PortalTls {
    fun spkiSha256(cert: X509Certificate): String = TODO("T24")

    /** [trustStore] null = platform default (tests pass a KeyStore with the mock CA). */
    fun socketFactory(host: String, pin: String, trustStore: KeyStore? = null): SSLSocketFactory = TODO("T24")
}
