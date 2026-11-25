package io.github.autowifibits.core

/** Persisted session, same fields as internal/state.SessionState. Times are epoch millis (0 = never). */
data class SessionState(
    val magicToken: String = "",
    val keepaliveUrl: String = "",
    val logoutUrl: String = "",
    val consecutiveFailures: Int = 0,
    val lastNotificationAtMs: Long = 0,
    val credentialsRejected: Boolean = false,
) {
    val loggedIn: Boolean get() = keepaliveUrl.isNotEmpty()
    fun clearSession(): SessionState = copy(magicToken = "", keepaliveUrl = "", logoutUrl = "")
}

enum class Trigger { START, TIMER, NETWORK, MANUAL, CREDS_UPDATED }

enum class NotifyReason(val body: String) {
    REJECTED("Campus Wi-Fi login was rejected. Update your ID and password to resume."),
    REPEATED_ERRORS("Campus Wi-Fi login keeps failing. Will keep retrying every 5 minutes."),
    FORM_CHANGED("The campus login page has changed. Please report it to the app's GitHub page."),
    NO_CREDENTIALS("Campus Wi-Fi needs a login. Set your ID and password in AutoWiFi BITS."),
}

/** What one run did. [outcome] is an [Outcome] wire name, or "keepalive_ok" / "paused". */
data class Step(val outcome: String, val nextMs: Long, val notified: NotifyReason? = null)

/** Platform side of the engine (Android: SharedPreferences, NotificationManager, Keystore store). */
interface EnginePorts {
    fun credentials(): Credentials?
    fun notify(title: String, body: String)
    fun now(): Long
    fun save(state: SessionState)
}

object Backoff {
    const val STEADY_MS = 5 * 60_000L
    /** 1 → 5 s, 2 → 15 s, 3 → 60 s, ≥4 → STEADY_MS, ≤0 → STEADY_MS. */
    fun delayMs(failures: Int): Long = when (failures) {
        1 -> 5_000L
        2 -> 15_000L
        3 -> 60_000L
        else -> STEADY_MS
    }
}

private const val NOTIFY_COOLDOWN_MS = 60 * 60_000L

/** At most one notification per hour; a "last" in the future also allows (clock moved back). */
fun notifyAllowed(lastMs: Long, nowMs: Long): Boolean =
    lastMs <= 0L || lastMs > nowMs || (nowMs - lastMs) >= NOTIFY_COOLDOWN_MS

/**
 * Same rules as internal/engine.Engine.Run (read its doc comment: rules 1-5 and "notify r"),
 * ported to Kotlin (T23). The title of every notification is "AutoWiFi BITS". Not thread-safe:
 * the Android worker runs one check at a time (unique work).
 */
class Engine(private val portal: PortalApi, private val ports: EnginePorts, var state: SessionState) {
    fun run(trigger: Trigger): Step {
        val step = execute(trigger)
        ports.save(state)
        return step
    }

    private fun execute(trigger: Trigger): Step {
        if (trigger == Trigger.CREDS_UPDATED) {
            state = state.copy(credentialsRejected = false, consecutiveFailures = 0)
        } else {
            if (state.credentialsRejected && trigger != Trigger.MANUAL) {
                return Step("paused", Backoff.STEADY_MS)
            }
            if (state.loggedIn && trigger == Trigger.TIMER) {
                if (portal.keepalive(state.keepaliveUrl)) {
                    state = state.copy(consecutiveFailures = 0)
                    return Step("keepalive_ok", Backoff.STEADY_MS)
                }
                state = state.clearSession()
            }
        }

        val r = portal.checkOnce { ports.credentials() }
        return when (r.outcome) {
            Outcome.OFF_CAMPUS -> {
                state = state.clearSession().copy(consecutiveFailures = 0)
                Step(r.outcome.wire, Backoff.STEADY_MS)
            }
            Outcome.ONLINE, Outcome.FOREIGN -> {
                state = state.copy(consecutiveFailures = 0)
                Step(r.outcome.wire, Backoff.STEADY_MS)
            }
            Outcome.LOGGED_IN -> {
                state = state.copy(
                    magicToken = r.magic,
                    keepaliveUrl = r.keepaliveUrl,
                    logoutUrl = r.logoutUrl,
                    consecutiveFailures = 0,
                    credentialsRejected = false,
                )
                Step(r.outcome.wire, Backoff.STEADY_MS)
            }
            Outcome.REJECTED -> {
                state = state.clearSession().copy(credentialsRejected = true, consecutiveFailures = 0)
                Step(r.outcome.wire, Backoff.STEADY_MS, notifyIfAllowed(NotifyReason.REJECTED))
            }
            Outcome.NO_CREDENTIALS -> {
                state = state.copy(consecutiveFailures = 0)
                Step(r.outcome.wire, Backoff.STEADY_MS, notifyIfAllowed(NotifyReason.NO_CREDENTIALS))
            }
            Outcome.FORM_CHANGED -> {
                val failures = state.consecutiveFailures + 1
                state = state.copy(consecutiveFailures = failures)
                Step(r.outcome.wire, Backoff.delayMs(failures), notifyIfAllowed(NotifyReason.FORM_CHANGED))
            }
            Outcome.PORTAL_ERROR -> {
                val failures = state.consecutiveFailures + 1
                state = state.copy(consecutiveFailures = failures)
                val notified = if (failures >= 3) notifyIfAllowed(NotifyReason.REPEATED_ERRORS) else null
                Step(r.outcome.wire, Backoff.delayMs(failures), notified)
            }
        }
    }

    private fun notifyIfAllowed(reason: NotifyReason): NotifyReason? {
        val now = ports.now()
        if (!notifyAllowed(state.lastNotificationAtMs, now)) return null
        state = state.copy(lastNotificationAtMs = now)
        ports.notify("AutoWiFi BITS", reason.body)
        return reason
    }

    fun logout() {
        if (state.loggedIn) {
            portal.logout(state.logoutUrl)
            state = state.clearSession()
        }
        ports.save(state)
    }
}

/** The part of [PortalClient] the engine uses (faked in tests). */
interface PortalApi {
    fun checkOnce(getCreds: () -> Credentials?): CheckResult
    fun keepalive(url: String): Boolean
    fun logout(url: String)
}
