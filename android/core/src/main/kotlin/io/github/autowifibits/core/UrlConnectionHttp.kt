package io.github.autowifibits.core

import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLConnection
import java.net.URLEncoder
import javax.net.ssl.HttpsURLConnection
import javax.net.ssl.SSLSocketFactory

/**
 * [Http] over HttpURLConnection (T22).
 * - [open] creates the connection: `URL::openConnection` on the JVM, `network::openConnection`
 *   on Android so requests go over the Wi-Fi network that triggered the check.
 * - [sslSocketFactory] is set on HttpsURLConnection when non-null (portal pinning, mock CA).
 * - instanceFollowRedirects = false, useCaches = false, connect timeout [timeoutMs], read timeout
 *   [timeoutMs], no proxy is configured by this class. Body read up to MAX_BODY bytes (error
 *   stream for status >= 400; empty when neither exists), decoded as UTF-8 with replacement.
 * - postForm encodes with URLEncoder.encode(.., UTF_8) ("+" for spaces) joined by "&".
 */
class UrlConnectionHttp(
    private val open: (URL) -> URLConnection = { it.openConnection() },
    private val sslSocketFactory: SSLSocketFactory? = null,
    private val timeoutMs: Int = 10_000,
) : Http {
    @Throws(IOException::class)
    override fun get(url: String): HttpReply = execute(url, "GET", body = null)

    @Throws(IOException::class)
    override fun postForm(url: String, fields: List<Pair<String, String>>): HttpReply {
        val encoded = fields.joinToString("&") { (k, v) ->
            "${URLEncoder.encode(k, "UTF-8")}=${URLEncoder.encode(v, "UTF-8")}"
        }
        return execute(url, "POST", body = encoded.toByteArray(Charsets.UTF_8))
    }

    private fun execute(url: String, method: String, body: ByteArray?): HttpReply {
        val conn = open(URL(url)) as HttpURLConnection
        try {
            if (conn is HttpsURLConnection && sslSocketFactory != null) {
                conn.sslSocketFactory = sslSocketFactory
            }
            conn.instanceFollowRedirects = false
            conn.useCaches = false
            conn.connectTimeout = timeoutMs
            conn.readTimeout = timeoutMs
            conn.requestMethod = method

            if (body != null) {
                conn.doOutput = true
                conn.setRequestProperty("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
                conn.outputStream.use { it.write(body) }
            }

            val status = conn.responseCode
            val location = conn.getHeaderField("Location") ?: ""
            val stream = if (status >= 400) conn.errorStream else conn.inputStream
            return HttpReply(status, location, readLimited(stream))
        } finally {
            conn.disconnect()
        }
    }

    private fun readLimited(stream: InputStream?): String {
        if (stream == null) return ""
        return stream.use { input ->
            val buf = ByteArray(minOf(8192, MAX_BODY))
            val out = ByteArrayOutputStream()
            var remaining = MAX_BODY
            while (remaining > 0) {
                val n = input.read(buf, 0, minOf(buf.size, remaining))
                if (n <= 0) break
                out.write(buf, 0, n)
                remaining -= n
            }
            String(out.toByteArray(), Charsets.UTF_8)
        }
    }
}
