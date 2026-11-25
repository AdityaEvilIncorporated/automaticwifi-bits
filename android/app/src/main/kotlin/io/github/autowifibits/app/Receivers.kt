package io.github.autowifibits.app

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import io.github.autowifibits.core.Trigger
import java.util.concurrent.TimeUnit
import kotlin.concurrent.thread

/**
 * Keeps the receiver alive (goAsync) until WorkManager has persisted the work: a cold-started
 * process may be frozen as soon as onReceive returns.
 */
private fun BroadcastReceiver.enqueueAndFinish(block: () -> Unit) {
    val pending = goAsync()
    thread(name = "autowifi-enqueue") {
        try {
            block()
        } finally {
            pending.finish()
        }
    }
}

/** Wi-Fi became available (PendingIntent network callback). */
class NetworkReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) = enqueueAndFinish {
        runCatching { Scheduler.now(context, Trigger.NETWORK).result.get(8, TimeUnit.SECONDS) }
    }
}

/** Re-register the network callback and periodic work after a reboot or an app update. */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED && intent.action != Intent.ACTION_MY_PACKAGE_REPLACED) return
        enqueueAndFinish {
            runCatching {
                Scheduler.ensure(context)
                Scheduler.now(context, Trigger.START).result.get(8, TimeUnit.SECONDS)
            }
        }
    }
}
