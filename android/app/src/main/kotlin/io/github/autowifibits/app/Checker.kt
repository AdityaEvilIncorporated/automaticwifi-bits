package io.github.autowifibits.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import io.github.autowifibits.core.Credentials
import io.github.autowifibits.core.Engine
import io.github.autowifibits.core.EnginePorts
import io.github.autowifibits.core.PortalClient
import io.github.autowifibits.core.PortalConfig
import io.github.autowifibits.core.PortalTls
import io.github.autowifibits.core.Resolver
import io.github.autowifibits.core.SessionState
import io.github.autowifibits.core.Step
import io.github.autowifibits.core.Trigger
import io.github.autowifibits.core.UrlConnectionHttp
import java.net.UnknownHostException

/**
 * One check, run by CheckWorker. Every request (probe, DNS, portal) goes over the Wi-Fi Network
 * itself: a captive Wi-Fi is not "validated", so the default route may be mobile data.
 */
object Checker {
    private val lock = Any()
    const val CHANNEL = "alerts"

    fun wifi(context: Context): Network? {
        val cm = context.getSystemService(ConnectivityManager::class.java)
        return cm.allNetworks.firstOrNull { n ->
            cm.getNetworkCapabilities(n)?.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) == true
        }
    }

    fun run(context: Context, trigger: Trigger): Step = synchronized(lock) {
        val app = context.applicationContext
        val states = StateStore(app)
        val store = CredentialStore(app)
        val net = wifi(app)
        val step: Step
        if (net == null) {
            // No Wi-Fi at all: nothing to log in to (SSIDs are never read).
            val s = states.load().clearSession().copy(consecutiveFailures = 0)
            states.save(s)
            step = Step("off_campus", io.github.autowifibits.core.Backoff.STEADY_MS)
        } else {
            val cfg = PortalConfig()
            val portal = PortalClient(
                cfg,
                probe = UrlConnectionHttp(open = { net.openConnection(it) }),
                portal = UrlConnectionHttp(open = { net.openConnection(it) },
                    sslSocketFactory = PortalTls.socketFactory(cfg.portalHost, cfg.pinnedCertSha256)),
                resolver = Resolver { host ->
                    try { net.getAllByName(host).isNotEmpty() } catch (_: UnknownHostException) { false }
                },
            )
            val ports = object : EnginePorts {
                override fun credentials(): Credentials? = store.get()
                override fun notify(title: String, body: String) = post(app, title, body)
                override fun now() = System.currentTimeMillis()
                override fun save(state: SessionState) = states.save(state)
            }
            step = Engine(portal, ports, states.load()).run(trigger)
        }
        states.lastOutcome = step.outcome
        Scheduler.afterStep(app, step)
        step
    }

    fun logout(context: Context) = synchronized(lock) {
        val app = context.applicationContext
        val states = StateStore(app)
        val net = wifi(app)
        val s = states.load()
        if (net != null && s.loggedIn) {
            val cfg = PortalConfig()
            val client = PortalClient(cfg, UrlConnectionHttp(open = { net.openConnection(it) }),
                UrlConnectionHttp(open = { net.openConnection(it) },
                    sslSocketFactory = PortalTls.socketFactory(cfg.portalHost, cfg.pinnedCertSha256)),
                Resolver { true })
            client.logout(s.logoutUrl)
        }
        states.save(s.clearSession())
        states.lastOutcome = "logged_out"
    }

    fun ensureChannel(context: Context) {
        val nm = context.getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(NotificationChannel(CHANNEL, "Login problems", NotificationManager.IMPORTANCE_LOW))
    }

    private fun post(context: Context, title: String, body: String) {
        ensureChannel(context)
        val n = Notification.Builder(context, CHANNEL)
            .setSmallIcon(android.R.drawable.stat_sys_warning)
            .setContentTitle(title)
            .setContentText(body)
            .setStyle(Notification.BigTextStyle().bigText(body))
            .setAutoCancel(true)
            .build()
        // Throws SecurityException without POST_NOTIFICATIONS on 13+; the Engine catches it.
        context.getSystemService(NotificationManager::class.java).notify(1, n)
    }
}
