package io.github.autowifibits.app

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import io.github.autowifibits.core.Trigger

/** Wi-Fi became available (PendingIntent network callback). */
class NetworkReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        Scheduler.now(context, Trigger.NETWORK)
    }
}

/** Re-register the network callback and periodic work after a reboot or an app update. */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Intent.ACTION_BOOT_COMPLETED || intent.action == Intent.ACTION_MY_PACKAGE_REPLACED) {
            Scheduler.ensure(context)
            Scheduler.now(context, Trigger.START)
        }
    }
}
