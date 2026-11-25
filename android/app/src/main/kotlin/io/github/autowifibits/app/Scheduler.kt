package io.github.autowifibits.app

import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import androidx.work.Data
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.Operation
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import io.github.autowifibits.core.Step
import io.github.autowifibits.core.Trigger
import java.util.concurrent.TimeUnit

/** All scheduling: the Wi-Fi callback, the 15-minute periodic check, and the next delayed check. */
object Scheduler {
    private const val PERIODIC = "periodic"
    private const val NEXT = "next"
    private const val NOW = "now"

    /** Idempotent; called from the activity, at boot and after an app update. */
    fun ensure(context: Context) {
        val app = context.applicationContext
        val cm = app.getSystemService(ConnectivityManager::class.java)
        val request = NetworkRequest.Builder().addTransportType(NetworkCapabilities.TRANSPORT_WIFI).build()
        // A PendingIntent callback survives the app process being killed (no service needed).
        // FLAG_MUTABLE: the system adds the Network to the intent's extras.
        val pi = PendingIntent.getBroadcast(app, 0, Intent(app, NetworkReceiver::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE)
        cm.registerNetworkCallback(request, pi)
        WorkManager.getInstance(app).enqueueUniquePeriodicWork(PERIODIC, ExistingPeriodicWorkPolicy.KEEP,
            PeriodicWorkRequestBuilder<CheckWorker>(15, TimeUnit.MINUTES)
                .setInputData(CheckWorker.input(Trigger.TIMER)).build())
    }

    /**
     * Run a check as soon as possible (Wi-Fi joined, button pressed, credentials saved). Each
     * trigger has its own unique work, so a Wi-Fi join can never cancel a pending
     * CREDS_UPDATED run (which is what clears the rejected flag).
     */
    fun now(context: Context, trigger: Trigger): Operation =
        WorkManager.getInstance(context.applicationContext).enqueueUniqueWork("$NOW-${trigger.name}", ExistingWorkPolicy.REPLACE,
            OneTimeWorkRequestBuilder<CheckWorker>().setInputData(CheckWorker.input(trigger)).build())

    fun cancelNext(context: Context) {
        WorkManager.getInstance(context.applicationContext).cancelUniqueWork(NEXT)
    }

    /**
     * After a check: keepalive in 5 min when logged in, retries at 5 s / 15 s / 60 s / 5 min after
     * errors. WorkManager may delay these under Doze; the periodic work is the safety net.
     */
    fun afterStep(context: Context, step: Step, loggedIn: Boolean) {
        val wm = WorkManager.getInstance(context.applicationContext)
        val delay = Logic.nextDelayMs(step, loggedIn)
        if (delay == null) {
            wm.cancelUniqueWork(NEXT) // periodic work only
        } else {
            wm.enqueueUniqueWork(NEXT, ExistingWorkPolicy.REPLACE,
                OneTimeWorkRequestBuilder<CheckWorker>()
                    .setInitialDelay(delay, TimeUnit.MILLISECONDS)
                    .setInputData(CheckWorker.input(Trigger.TIMER)).build())
        }
    }
}

/** Data helper kept here so the worker file stays small. */
internal fun triggerData(t: Trigger): Data = Data.Builder().putString("trigger", t.name).build()
