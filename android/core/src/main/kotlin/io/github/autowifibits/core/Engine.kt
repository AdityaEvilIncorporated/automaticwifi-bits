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
    fun delayMs(failures: Int): Long = TODO("T23")
}

/** At most one notification per hour; a "last" in the future also allows (clock moved back). */
fun notifyAllowed(lastMs: Long, nowMs: Long): Boolean = TODO("T23")

/**
 * Same rules as internal/engine.Engine.Run (read its doc comment: rules 1-5 and "notify r"),
 * ported to Kotlin (T23). The title of every notification is "AutoWiFi BITS". Not thread-safe:
 * the Android worker runs one check at a time (unique work).
 */
class Engine(private val portal: PortalApi, private val ports: EnginePorts, var state: SessionState) {
    fun run(trigger: Trigger): Step = TODO("T23")
    fun logout(): Unit = TODO("T23")
}

/** The part of [PortalClient] the engine uses (faked in tests). */
interface PortalApi {
    fun checkOnce(getCreds: () -> Credentials?): CheckResult
    fun keepalive(url: String): Boolean
    fun logout(url: String)
}
