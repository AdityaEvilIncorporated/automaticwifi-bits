package io.github.autowifibits.core

import org.json.JSONObject
import org.junit.jupiter.api.AfterAll
import org.junit.jupiter.api.Assumptions.assumeTrue
import org.junit.jupiter.api.BeforeAll
import org.junit.jupiter.api.TestInstance
import java.io.File
import java.net.URLEncoder
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/** protocol/testcases/scenarios.json against the Go mock portal, plus keepalive/logout. */
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class ScenariosTest {
    private lateinit var mock: MockPortal

    @BeforeAll fun up() { assumeTrue(MockPortal.available(), "MOCKPORTAL_BIN not set"); mock = MockPortal.start() }
    @AfterAll fun down() { if (::mock.isInitialized) mock.close() }

    private fun client() = PortalClient(
        mock.config(), UrlConnectionHttp(), UrlConnectionHttp(sslSocketFactory = mock.sslFactory()), Resolver { true },
    )

    private val good = Credentials("f20250001", "correct-horse")

    @Test fun scenarios() {
        val dir = System.getProperty("testcases.dir")
        val arr = JSONObject(File(dir, "scenarios.json").readText()).getJSONArray("cases")
        for (i in 0 until arr.length()) {
            val c = arr.getJSONObject(i)
            val name = c.getString("name")
            val m = c.getJSONObject("mock")
            mock.control("reset")
            if (m.has("style")) mock.control("style?probe=" + m.getString("style"))
            if (m.optBoolean("authenticated")) mock.control("auth?value=1")
            if (m.has("fail")) mock.control("fail?n=" + m.getInt("fail"))
            if (m.has("fields")) {
                val f = m.getJSONObject("fields")
                mock.control("fields?" + f.keys().asSequence().joinToString("&") { "$it=" + URLEncoder.encode(f.getString(it), "UTF-8") })
            }
            val cr = if (c.isNull("credentials")) null else c.getJSONObject("credentials").let { Credentials(it.getString("username"), it.getString("password")) }
            val r = client().checkOnce { cr }
            val e = c.getJSONObject("expect")
            assertEquals(e.getString("outcome"), r.outcome.wire, "$name (error ${r.error})")
            val st = mock.control("state")
            val counters = e.getJSONObject("counters")
            for (k in counters.keys()) assertEquals(counters.getInt(k), st.getInt(k), "$name counter $k")
            if (r.outcome == Outcome.LOGGED_IN) {
                assertTrue(r.keepaliveUrl.contains("/keepalive?") && r.logoutUrl.contains("/logout?") && r.magic.isNotEmpty(), name)
            }
        }
    }

    @Test fun offCampus() {
        mock.control("reset")
        var asked = false
        val c = PortalClient(mock.config().copy(portalHost = "fw.bits-pilani.invalid"), UrlConnectionHttp(), UrlConnectionHttp(), Resolver { false })
        assertEquals(Outcome.OFF_CAMPUS, c.checkOnce { asked = true; good }.outcome)
        assertFalse(asked)
        assertEquals(0, mock.control("state").getInt("probe_hits"))
    }

    @Test fun keepaliveLogoutAndForeignUrls() {
        mock.control("reset")
        val c = client()
        val r = c.checkOnce { good }
        assertEquals(Outcome.LOGGED_IN, r.outcome, r.error)
        assertTrue(c.keepalive(r.keepaliveUrl))
        assertFalse(c.keepalive(r.keepaliveUrl.replace("127.0.0.1", "evil.example")))
        assertEquals(1, mock.control("state").getInt("keepalives"))
        c.logout(r.logoutUrl)
        assertFalse(mock.control("state").getBoolean("authenticated"))
        assertFalse(c.keepalive(r.keepaliveUrl))

        val foreign = ProbeResult(ProbeKind.CAPTIVE, "abc", "https://evil.example:${mock.portalPort}/fgtauth?abc")
        assertEquals(Outcome.FOREIGN, c.login(foreign, good).outcome)
    }

    @Test fun fallbackProbe() {
        mock.control("reset")
        val c = PortalClient(mock.config().copy(checkUrl = "http://127.0.0.1:1/"), UrlConnectionHttp(), UrlConnectionHttp(sslSocketFactory = mock.sslFactory()), Resolver { true })
        assertEquals(ProbeKind.CAPTIVE, c.probeOnce().kind)
    }

    @Test fun errorNeverContainsPassword() {
        mock.control("reset")
        mock.control("fail?n=5")
        val r = client().checkOnce { Credentials("f20250001", "S3cret-Pass") }
        assertEquals(Outcome.PORTAL_ERROR, r.outcome)
        assertFalse((r.error ?: "").contains("S3cret-Pass"))
    }
}
