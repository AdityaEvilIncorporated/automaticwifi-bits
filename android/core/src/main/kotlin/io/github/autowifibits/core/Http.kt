package io.github.autowifibits.core

import java.io.IOException

/** One HTTP reply. [location] is the Location header or "". [body] is at most MAX_BODY bytes, UTF-8 decoded. */
data class HttpReply(val status: Int, val location: String, val body: String)

/** Minimal HTTP used by [PortalClient]. Implementations never follow redirects. */
interface Http {
    /** GET [url]. Throws IOException on network/TLS errors; any HTTP status is a normal reply. */
    @Throws(IOException::class)
    fun get(url: String): HttpReply

    /** POST [fields] in order as application/x-www-form-urlencoded (UTF-8). Same error rules. */
    @Throws(IOException::class)
    fun postForm(url: String, fields: List<Pair<String, String>>): HttpReply
}

const val MAX_BODY = 256 * 1024

/** DNS lookup for the campus test; returns false when the host does not resolve. */
fun interface Resolver {
    fun resolves(host: String): Boolean
}
