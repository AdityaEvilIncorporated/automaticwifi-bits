package io.github.autowifibits.core

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

/** Mirrors internal/engine/engine_test.go. */
class EngineTest {
    private val ka = "https://fw.bits-pilani.ac.in:8090/keepalive?aa"
    private val lo = "https://fw.bits-pilani.ac.in:8090/logout?aa"
    private val loggedIn = CheckResult(Outcome.LOGGED_IN, "m1", ka, lo)

    private class FakePortal(var results: List<CheckResult>) : PortalApi {
        var checks = 0; var alive = false
        val keepalives = mutableListOf<String>(); val logouts = mutableListOf<String>()
        override fun checkOnce(getCreds: () -> Credentials?): CheckResult { getCreds(); checks++; return results[minOf(checks - 1, results.size - 1)] }
        override fun keepalive(url: String): Boolean { keepalives += url; return alive }
        override fun logout(url: String) { logouts += url }
    }

    private class Ports : EnginePorts {
        var nowMs = 1_800_000_000_000L
        val msgs = mutableListOf<String>(); var saves = 0
        override fun credentials() = Credentials("f20250001", "pw")
        override fun notify(title: String, body: String) { assertEquals("AutoWiFi BITS", title); msgs += body }
        override fun now() = nowMs
        override fun save(state: SessionState) { saves++ }
    }

    private fun rig(vararg r: CheckResult): Triple<Engine, FakePortal, Ports> {
        val p = FakePortal(r.toList()); val ports = Ports()
        return Triple(Engine(p, ports, SessionState()), p, ports)
    }

    @Test fun backoffAndLimit() {
        assertEquals(listOf(300_000L, 300_000L, 5_000L, 15_000L, 60_000L, 300_000L, 300_000L), listOf(-1, 0, 1, 2, 3, 4, 99).map(Backoff::delayMs))
        val now = 10_000_000L
        assertTrue(notifyAllowed(0, now)); assertFalse(notifyAllowed(now - 59 * 60_000, now))
        assertTrue(notifyAllowed(now - 60 * 60_000, now)); assertTrue(notifyAllowed(now + 1, now))
    }

    @Test fun loginThenKeepalive() {
        val (e, p, ports) = rig(loggedIn)
        assertEquals(Step("logged_in", Backoff.STEADY_MS), e.run(Trigger.START))
        assertEquals(ka, e.state.keepaliveUrl); assertEquals("m1", e.state.magicToken)
        p.alive = true
        assertEquals(Step("keepalive_ok", Backoff.STEADY_MS), e.run(Trigger.TIMER))
        assertEquals(1, p.checks); assertEquals(listOf(ka), p.keepalives); assertEquals(2, ports.saves)
        assertTrue(ports.msgs.isEmpty())
    }

    @Test fun keepaliveFailureRelogs() {
        val (e, p, _) = rig(loggedIn, CheckResult(Outcome.ONLINE))
        e.run(Trigger.START)
        assertEquals("online", e.run(Trigger.TIMER).outcome)
        assertEquals(2, p.checks); assertFalse(e.state.loggedIn)
    }

    @Test fun portalErrorsBackOffAndNotifyOnThird() {
        val (e, _, ports) = rig(CheckResult(Outcome.PORTAL_ERROR, error = "timeout"))
        val steps = (1..5).map { e.run(Trigger.TIMER).also { s -> ports.nowMs += s.nextMs } }
        assertEquals(listOf(5_000L, 15_000L, 60_000L, 300_000L, 300_000L), steps.map { it.nextMs })
        assertEquals(listOf(null, null, NotifyReason.REPEATED_ERRORS, null, null), steps.map { it.notified })
        assertEquals(listOf(NotifyReason.REPEATED_ERRORS.body), ports.msgs)
        assertEquals(5, e.state.consecutiveFailures)
    }

    @Test fun rejectedPausesUntilCredsUpdated() {
        val (e, p, ports) = rig(CheckResult(Outcome.REJECTED))
        assertEquals(Step("rejected", Backoff.STEADY_MS, NotifyReason.REJECTED), e.run(Trigger.START))
        for (t in listOf(Trigger.TIMER, Trigger.NETWORK, Trigger.START)) assertEquals(Step("paused", Backoff.STEADY_MS), e.run(t))
        assertEquals(1, p.checks)
        val manual = e.run(Trigger.MANUAL)
        assertEquals("rejected", manual.outcome); assertNull(manual.notified); assertEquals(2, p.checks)
        p.results = listOf(loggedIn); p.checks = 0
        assertEquals("logged_in", e.run(Trigger.CREDS_UPDATED).outcome)
        assertFalse(e.state.credentialsRejected)
        assertEquals(1, ports.msgs.size)
    }

    @Test fun offCampusAndForeignAreQuiet() {
        val (e, _, ports) = rig(loggedIn, CheckResult(Outcome.OFF_CAMPUS), CheckResult(Outcome.FOREIGN))
        e.run(Trigger.START)
        e.state = e.state.copy(consecutiveFailures = 2)
        assertEquals(Step("off_campus", Backoff.STEADY_MS), e.run(Trigger.NETWORK))
        assertFalse(e.state.loggedIn); assertEquals(0, e.state.consecutiveFailures)
        assertEquals(Step("foreign", Backoff.STEADY_MS), e.run(Trigger.NETWORK))
        assertTrue(ports.msgs.isEmpty())
    }

    @Test fun formChangedAndNoCredentials() {
        val (e, _, ports) = rig(CheckResult(Outcome.FORM_CHANGED))
        assertEquals(Step("form_changed", 5_000L, NotifyReason.FORM_CHANGED), e.run(Trigger.START))
        assertEquals(Step("form_changed", 15_000L), e.run(Trigger.TIMER))
        val (e2, _, _) = rig(CheckResult(Outcome.NO_CREDENTIALS))
        assertEquals(Step("no_credentials", Backoff.STEADY_MS, NotifyReason.NO_CREDENTIALS), e2.run(Trigger.START))
        assertEquals(1, ports.msgs.size)
    }

    @Test fun logout() {
        val (e, p, _) = rig(loggedIn)
        e.run(Trigger.START)
        e.logout()
        assertEquals(listOf(lo), p.logouts); assertFalse(e.state.loggedIn)
        e.logout()
        assertEquals(1, p.logouts.size)
    }
}

class EngineNotifierFailureTest {
    @Test fun throwingNotifierStillSaves() {
        var saved: SessionState? = null
        val portal = object : PortalApi {
            override fun checkOnce(getCreds: () -> Credentials?) = CheckResult(Outcome.REJECTED)
            override fun keepalive(url: String) = false
            override fun logout(url: String) {}
        }
        val ports = object : EnginePorts {
            override fun credentials() = Credentials("a", "b")
            override fun notify(title: String, body: String) = throw SecurityException("no permission")
            override fun now() = 5_000_000L
            override fun save(state: SessionState) { saved = state }
        }
        val step = Engine(portal, ports, SessionState()).run(Trigger.START)
        assertEquals(NotifyReason.REJECTED, step.notified)
        assertTrue(saved!!.credentialsRejected)
        assertEquals(5_000_000L, saved!!.lastNotificationAtMs)
    }
}
