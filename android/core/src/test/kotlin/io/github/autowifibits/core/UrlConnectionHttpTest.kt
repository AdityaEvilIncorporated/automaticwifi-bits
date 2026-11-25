package io.github.autowifibits.core

import com.sun.net.httpserver.HttpServer
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.BeforeEach
import java.io.IOException
import java.net.InetSocketAddress
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

class UrlConnectionHttpTest {
    private lateinit var srv: HttpServer
    private var lastBody = ""
    private var lastType = ""
    private val base get() = "http://127.0.0.1:${srv.address.port}"

    @BeforeEach fun up() {
        srv = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        srv.createContext("/redir") { ex -> ex.responseHeaders.add("Location", "https://fw.example:8090/fgtauth?ab"); ex.sendResponseHeaders(303, -1); ex.close() }
        srv.createContext("/err") { ex -> val b = "boom".toByteArray(); ex.sendResponseHeaders(500, b.size.toLong()); ex.responseBody.write(b); ex.close() }
        srv.createContext("/big") { ex -> ex.sendResponseHeaders(200, 0); ex.responseBody.write(ByteArray(MAX_BODY + 1000) { 'a'.code.toByte() }); ex.close() }
        srv.createContext("/204") { ex -> ex.sendResponseHeaders(204, -1); ex.close() }
        srv.createContext("/post") { ex ->
            lastType = ex.requestHeaders.getFirst("Content-Type") ?: ""
            lastBody = ex.requestBody.readBytes().toString(Charsets.UTF_8)
            val b = "ok é".toByteArray(Charsets.UTF_8); ex.sendResponseHeaders(200, b.size.toLong()); ex.responseBody.write(b); ex.close()
        }
        srv.start()
    }

    @AfterEach fun down() = srv.stop(0)

    @Test fun doesNotFollowRedirects() {
        assertEquals(HttpReply(303, "https://fw.example:8090/fgtauth?ab", ""), UrlConnectionHttp().get("$base/redir"))
    }

    @Test fun errorBodyIsRead() = assertEquals(HttpReply(500, "", "boom"), UrlConnectionHttp().get("$base/err"))

    @Test fun noContent() = assertEquals(HttpReply(204, "", ""), UrlConnectionHttp().get("$base/204"))

    @Test fun bodyIsCapped() = assertEquals(MAX_BODY, UrlConnectionHttp().get("$base/big").body.length)

    @Test fun postFormEncodesInOrder() {
        val r = UrlConnectionHttp().postForm("$base/post", listOf("username" to "f2025 01", "password" to "p&ss=wörd", "magic" to "m1", "4Tredir" to "http://x/?a=1"))
        assertEquals("ok é", r.body)
        assertEquals("username=f2025+01&password=p%26ss%3Dw%C3%B6rd&magic=m1&4Tredir=http%3A%2F%2Fx%2F%3Fa%3D1", lastBody)
        assertEquals("application/x-www-form-urlencoded", lastType.substringBefore(";"))
    }

    @Test fun networkErrorThrows() {
        assertFailsWith<IOException> { UrlConnectionHttp(timeoutMs = 2000).get("http://127.0.0.1:1/") }
    }
}
