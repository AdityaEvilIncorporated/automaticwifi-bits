package io.github.autowifibits.app

import io.github.autowifibits.core.CheckResult
import io.github.autowifibits.core.Credentials
import io.github.autowifibits.core.Outcome
import io.github.autowifibits.core.PortalApi
import io.github.autowifibits.core.Step

/** Pure decisions used by the Android glue, kept here so they are unit-tested on the JVM. */
object Logic {
    /**
     * Delay before the next one-time check after [step], or null for "periodic work only".
     * [loggedIn] is the session state after the step: while a session is held (e.g. a Wi-Fi
     * roam or "Log in now" found the network online) the 5-minute keepalive must stay scheduled.
     */
    fun nextDelayMs(step: Step, loggedIn: Boolean): Long? = when {
        step.outcome in setOf("logged_in", "keepalive_ok", "portal_error", "form_changed") -> step.nextMs
        loggedIn && step.outcome != "paused" -> step.nextMs
        else -> null // online without a session, off campus, paused, rejected, no credentials, foreign
    }

    private val pinRe = Regex("^[0-9a-f]{64}$")

    /** Normalises a pasted pin (trim, lower-case); "" clears it. Throws for anything else. */
    fun normalizePin(v: String): String {
        val clean = v.trim().lowercase()
        require(clean.isEmpty() || pinRe.matches(clean)) { "The pin must be 64 hex characters." }
        return clean
    }

    /** Status line wording for the screen. */
    fun label(o: String): String = when (o) {
        "logged_in", "keepalive_ok" -> "logged in"
        "online" -> "already online"
        "off_campus" -> "not on the campus network"
        "foreign" -> "another network's login page"
        "rejected", "paused" -> "ID or password rejected"
        "no_credentials" -> "ID and password needed"
        "form_changed" -> "the login page changed"
        "portal_error" -> "portal not responding, retrying"
        "logged_out" -> "logged out"
        else -> o
    }
}

/**
 * The portal side when there is no Wi-Fi network: every check is "off campus". Running the real
 * Engine with it (instead of skipping the engine) keeps every rule, e.g. CREDS_UPDATED still
 * clears the rejected flag while the phone is on mobile data.
 */
object NoWifiPortal : PortalApi {
    override fun checkOnce(getCreds: () -> Credentials?) = CheckResult(Outcome.OFF_CAMPUS)
    override fun keepalive(url: String) = false
    override fun logout(url: String) {}
}
