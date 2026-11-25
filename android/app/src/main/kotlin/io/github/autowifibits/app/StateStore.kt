package io.github.autowifibits.app

import android.annotation.SuppressLint
import android.content.Context
import io.github.autowifibits.core.SessionState

/**
 * SessionState in app-private SharedPreferences (the keepalive URL carries a session token).
 * Writes use commit(): they run on the worker thread, and the process may be frozen as soon as
 * doWork returns, so an asynchronous apply() could lose them.
 */
@SuppressLint("ApplySharedPref")
class StateStore(context: Context) {
    private val p = context.getSharedPreferences("state", Context.MODE_PRIVATE)

    fun load() = SessionState(
        magicToken = p.getString("magic_token", "") ?: "",
        keepaliveUrl = p.getString("keepalive_url", "") ?: "",
        logoutUrl = p.getString("logout_url", "") ?: "",
        consecutiveFailures = p.getInt("consecutive_failures", 0),
        lastNotificationAtMs = p.getLong("last_notification_at", 0),
        credentialsRejected = p.getBoolean("credentials_rejected", false),
    )

    fun save(s: SessionState) {
        p.edit()
            .putString("magic_token", s.magicToken)
            .putString("keepalive_url", s.keepaliveUrl)
            .putString("logout_url", s.logoutUrl)
            .putInt("consecutive_failures", s.consecutiveFailures)
            .putLong("last_notification_at", s.lastNotificationAtMs)
            .putBoolean("credentials_rejected", s.credentialsRejected)
            .commit() // synchronous: the worker process may be frozen right after doWork returns
    }

    var lastOutcome: String
        get() = p.getString("last_outcome", "") ?: ""
        set(v) { p.edit().putString("last_outcome", v).putLong("last_outcome_at", System.currentTimeMillis()).commit() }
}
