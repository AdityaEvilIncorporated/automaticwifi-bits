package io.github.autowifibits.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import io.github.autowifibits.core.Credentials
import io.github.autowifibits.core.Engine
import io.github.autowifibits.core.EnginePorts
import io.github.autowifibits.core.PortalClient
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
        // The physical Wi-Fi: not a VPN running over it, and one that offers internet (a captive
        // network has INTERNET but not VALIDATED; Wi-Fi Direct / local-only hotspots have neither).
        return cm.allNetworks.firstOrNull { n ->
            val c = cm.getNetworkCapabilities(n) ?: return@firstOrNull false
            c.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) &&
                c.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN) &&
                c.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
        }
    }

    fun run(context: Context, trigger: Trigger): Step = synchronized(lock) {
        val app = context.applicationContext
        val states = StateStore(app)
        val store = CredentialStore(app)
        val net = wifi(app)
        val ports = object : EnginePorts {
            override fun credentials(): Credentials? = store.get()
            override fun notify(title: String, body: String) = post(app, title, body)
            override fun now() = System.currentTimeMillis()
            override fun save(state: SessionState) = states.save(state)
        }
        val engine: Engine
        if (net == null) {
            // No Wi-Fi: every check is off campus, but through the Engine so its rules still apply.
            engine = Engine(NoWifiPortal, ports, states.load())
        } else {
            val cfg = ConfigStore(app).load()
            val portal = PortalClient(
                cfg,
                probe = UrlConnectionHttp(open = { net.openConnection(it) }),
                portal = UrlConnectionHttp(open = { net.openConnection(it) },
                    sslSocketFactory = PortalTls.socketFactory(cfg.portalHost, cfg.pinnedCertSha256)),
                resolver = Resolver { host ->
                    try { net.getAllByName(host).isNotEmpty() } catch (_: UnknownHostException) { false }
                },
            )
            engine = Engine(portal, ports, states.load())
        }
        val step = engine.run(trigger)
        states.lastOutcome = step.outcome
        Scheduler.afterStep(app, step, engine.state.loggedIn)
        step
    }

    fun logout(context: Context) = synchronized(lock) {
        val app = context.applicationContext
        val states = StateStore(app)
        val net = wifi(app)
        val s = states.load()
        Scheduler.cancelNext(app) // a queued keepalive must not log straight back in
        if (net != null && s.loggedIn) {
            val cfg = ConfigStore(app).load()
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
            .setContentIntent(PendingIntent.getActivity(context, 0, Intent(context, MainActivity::class.java),
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT))
            .build()
        // Throws SecurityException without POST_NOTIFICATIONS on 13+; the Engine catches it.
        context.getSystemService(NotificationManager::class.java).notify(1, n)
    }
}
