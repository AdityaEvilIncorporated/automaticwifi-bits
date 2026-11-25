package io.github.autowifibits.core

import org.json.JSONObject
import java.io.File
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

/** Runs protocol/testcases/{probe,login_page,login_reply,keepalive}.json, same as the Go tests. */
class SharedCasesTest {
    private fun cases(file: String): List<JSONObject> {
        val dir = System.getProperty("testcases.dir") ?: error("testcases.dir not set")
        val arr = JSONObject(File(dir, file).readText()).getJSONArray("cases")
        return (0 until arr.length()).map { arr.getJSONObject(it) }
    }

    private fun cfg(c: JSONObject): PortalConfig =
        if (c.has("config")) PortalConfig.fromOverrides(c.getJSONObject("config").toMap()) else PortalConfig()

    @Test fun probe() {
        val all = cases("probe.json")
        assertTrue(all.isNotEmpty())
        for (c in all) {
            val cfg = cfg(c)
            val got = Classify.classifyProbe(cfg, c.getInt("status"), c.getString("location"), c.getString("body"))
            val e = c.getJSONObject("expect")
            val name = c.getString("name")
            assertEquals(e.getString("kind"), got.kind.wire, name)
            assertEquals(e.getString("magic"), got.magic, name)
            val wantUrl = if (got.kind == ProbeKind.CAPTIVE) cfg.portalBase() + "/fgtauth?" + got.magic else ""
            assertEquals(wantUrl, got.magicUrl, name)
        }
    }

    @Test fun loginPage() {
        for (c in cases("login_page.json")) {
            val e = c.getJSONObject("expect")
            val name = c.getString("name")
            if (!e.getBoolean("ok")) {
                assertFailsWith<FormChangedException>(name) { Classify.parseLoginPage(cfg(c), c.getString("body"), c.getString("probe_magic")) }
                continue
            }
            val got = Classify.parseLoginPage(cfg(c), c.getString("body"), c.getString("probe_magic"))
            assertEquals(e.getString("magic"), got.magic, name)
            assertEquals(e.getString("redirect"), got.redirect, name)
        }
    }

    @Test fun loginReply() {
        for (c in cases("login_reply.json")) {
            val e = c.getJSONObject("expect")
            val got = Classify.classifyLoginReply(cfg(c), c.getInt("status"), c.getString("location"), c.getString("body"))
            assertEquals(
                LoginReply(Outcome.fromWire(e.getString("outcome")), e.getString("keepalive_url"), e.getString("logout_url")),
                got, c.getString("name"),
            )
        }
    }

    @Test fun keepalive() {
        for (c in cases("keepalive.json")) {
            assertEquals(c.getJSONObject("expect").getBoolean("alive"),
                Classify.keepaliveAlive(PortalConfig(), c.getInt("status"), c.getString("body")), c.getString("name"))
        }
    }

    @Test fun onPortal() {
        val cfg = PortalConfig()
        assertTrue(Classify.onPortal(cfg, "https://FW.bits-pilani.ac.in:8090/keepalive?1"))
        for (u in listOf("http://fw.bits-pilani.ac.in:8090/x", "https://fw.bits-pilani.ac.in/x",
            "https://fw.bits-pilani.ac.in.evil.example:8090/x", "nonsense", "")) {
            assertTrue(!Classify.onPortal(cfg, u), u)
        }
        val c443 = cfg.copy(portalPort = 443)
        assertTrue(Classify.onPortal(c443, "https://fw.bits-pilani.ac.in/x"))
    }

    @Test fun configAndCredentials() {
        assertEquals("https://fw.bits-pilani.ac.in:8090", PortalConfig().portalBase())
        assertEquals("https://fw.x.org:443", PortalConfig(portalHost = "FW.X.org", portalPort = 443).portalBase())
        val o = PortalConfig.fromOverrides(mapOf("portal_host" to "127.0.0.1", "portal_port" to 9443, "username_field" to "uid", "bogus" to 1))
        assertEquals(PortalConfig(portalHost = "127.0.0.1", portalPort = 9443, usernameField = "uid"), o)
        val cr = Credentials("f20250001", "hunter2")
        assertEquals("Credentials(username=f20250001, password=[redacted])", cr.toString())
        assertEquals("Credentials(username=a, password=[])", Credentials("a", "").toString())
    }
}
