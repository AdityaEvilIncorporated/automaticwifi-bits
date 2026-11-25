package io.github.autowifibits.app

import io.github.autowifibits.core.CheckResult
import io.github.autowifibits.core.Credentials
import io.github.autowifibits.core.Outcome
import io.github.autowifibits.core.PortalApi
import io.github.autowifibits.core.Step

/** Pure decisions used by the Android glue, kept here so they are unit-tested on the JVM. */
object Logic {
    /** Delay before the next one-time check after [step], or null for "periodic work only". */
    fun nextDelayMs(step: Step): Long? = when (step.outcome) {
        "logged_in", "keepalive_ok", "portal_error", "form_changed" -> step.nextMs
        else -> null // online, off campus, paused, rejected, no credentials, foreign
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
