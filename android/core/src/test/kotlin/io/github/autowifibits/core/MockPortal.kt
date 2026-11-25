package io.github.autowifibits.core

import org.json.JSONObject
import java.io.File
import java.net.HttpURLConnection
import java.net.ServerSocket
import java.net.URL
import java.security.KeyStore
import java.security.cert.CertificateFactory
import javax.net.ssl.SSLContext
import javax.net.ssl.TrustManagerFactory

/** Starts the Go mock portal binary ($MOCKPORTAL_BIN) for one test class. */
class MockPortal private constructor(private val proc: Process, val probeBase: String, val portalHost: String, val portalPort: Int, val caPem: File, val goSpki: String) : AutoCloseable {
    companion object {
        fun available(): Boolean = System.getenv("MOCKPORTAL_BIN")?.let { File(it).canExecute() } == true

        private fun freePort(): Int = ServerSocket(0).use { it.localPort }

        fun start(): MockPortal {
            val bin = System.getenv("MOCKPORTAL_BIN") ?: error("MOCKPORTAL_BIN not set")
            val probe = freePort(); val portal = freePort()
            val ca = File.createTempFile("mock-ca", ".pem")
            val p = ProcessBuilder(bin, "-probe", "127.0.0.1:$probe", "-portal", "127.0.0.1:$portal", "-ca-out", ca.path)
                .redirectError(ProcessBuilder.Redirect.INHERIT).start()
            val line = p.inputStream.bufferedReader().readLine() ?: error("mockportal exited")
            val cfg = JSONObject(line)
            return MockPortal(p, "http://127.0.0.1:$probe", cfg.getString("portal_host"), cfg.getInt("portal_port"), ca, cfg.getString("pinned_cert_sha256"))
        }
    }

    fun config(): PortalConfig = PortalConfig(checkUrl = "$probeBase/", fallbackCheckUrl = "$probeBase/generate_204", portalHost = portalHost, portalPort = portalPort)

    fun trustStore(): KeyStore {
        val ks = KeyStore.getInstance(KeyStore.getDefaultType()); ks.load(null)
        ks.setCertificateEntry("mock", CertificateFactory.getInstance("X.509").generateCertificate(caPem.inputStream()))
        return ks
    }

    /** SSL factory trusting only the mock CA (no pinning logic: that is PortalTls, T24). */
    fun sslFactory(): javax.net.ssl.SSLSocketFactory {
        val tmf = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm()); tmf.init(trustStore())
        return SSLContext.getInstance("TLS").apply { init(null, tmf.trustManagers, null) }.socketFactory
    }

    /** GET /__mock/<path> and return the state JSON. */
    fun control(path: String): JSONObject {
        val c = URL("$probeBase/__mock/$path").openConnection() as HttpURLConnection
        return JSONObject(c.inputStream.bufferedReader().readText())
    }

    fun cert(): java.security.cert.X509Certificate =
        CertificateFactory.getInstance("X.509").generateCertificate(caPem.inputStream()) as java.security.cert.X509Certificate

    override fun close() { proc.destroy(); caPem.delete() }
}
