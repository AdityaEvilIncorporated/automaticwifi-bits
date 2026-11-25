package io.github.autowifibits.core

import java.net.URL
import java.net.URLConnection
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
    override fun get(url: String): HttpReply = TODO("T22")
    override fun postForm(url: String, fields: List<Pair<String, String>>): HttpReply = TODO("T22")
}
