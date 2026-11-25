package io.github.autowifibits.app

import io.github.autowifibits.core.Backoff
import io.github.autowifibits.core.Credentials
import io.github.autowifibits.core.Engine
import io.github.autowifibits.core.EnginePorts
import io.github.autowifibits.core.SessionState
import io.github.autowifibits.core.Step
import io.github.autowifibits.core.Trigger
import org.junit.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

class LogicTest {
    @Test fun nextDelay() {
        assertEquals(300_000L, Logic.nextDelayMs(Step("logged_in", 300_000L), loggedIn = true))
        assertEquals(300_000L, Logic.nextDelayMs(Step("keepalive_ok", 300_000L), loggedIn = true))
        assertEquals(5_000L, Logic.nextDelayMs(Step("portal_error", 5_000L), loggedIn = false))
        assertEquals(15_000L, Logic.nextDelayMs(Step("form_changed", 15_000L), loggedIn = false))
        for (o in listOf("online", "off_campus", "foreign", "paused", "rejected", "no_credentials")) {
            assertNull(Logic.nextDelayMs(Step(o, Backoff.STEADY_MS), loggedIn = false), o)
        }
        // a Wi-Fi roam / "Log in now" that finds the network online keeps the keepalive going
        assertEquals(Backoff.STEADY_MS, Logic.nextDelayMs(Step("online", Backoff.STEADY_MS), loggedIn = true))
        assertNull(Logic.nextDelayMs(Step("paused", Backoff.STEADY_MS), loggedIn = true))
    }

    @Test fun pinNormalisation() {
        val pin = "ab".repeat(32)
        assertEquals(pin, Logic.normalizePin("  " + pin.uppercase() + "\n"))
        assertEquals("", Logic.normalizePin("   "))
        for (bad in listOf("abc", "g".repeat(64), "ab:".repeat(21) + "a")) {
            kotlin.test.assertFailsWith<IllegalArgumentException>(bad) { Logic.normalizePin(bad) }
        }
    }

    @Test fun everyOutcomeHasWording() {
        for (o in listOf("logged_in", "keepalive_ok", "online", "off_campus", "foreign", "rejected", "paused",
            "no_credentials", "form_changed", "portal_error", "logged_out")) {
            assertTrue(Logic.label(o) != o, o)
        }
    }

    private class Ports(var s: SessionState) : EnginePorts {
        override fun credentials() = Credentials("f2025", "pw")
        override fun notify(title: String, body: String) {}
        override fun now() = 1L
        override fun save(state: SessionState) { s = state }
    }

    /** Regression: saving new credentials while off Wi-Fi must clear the rejected flag. */
    @Test fun credsUpdatedOffWifiUnpauses() {
        val ports = Ports(SessionState(credentialsRejected = true, keepaliveUrl = "https://fw.bits-pilani.ac.in:8090/keepalive?1"))
        val e = Engine(NoWifiPortal, ports, ports.s)
        assertEquals("paused", e.run(Trigger.TIMER).outcome)
        val step = e.run(Trigger.CREDS_UPDATED)
        assertEquals("off_campus", step.outcome)
        assertFalse(ports.s.credentialsRejected)
        assertFalse(ports.s.loggedIn, "off campus drops the session")
    }
}
