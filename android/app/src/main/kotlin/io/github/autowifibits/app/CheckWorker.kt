package io.github.autowifibits.app

import android.content.Context
import androidx.work.Data
import androidx.work.Worker
import androidx.work.WorkerParameters
import io.github.autowifibits.core.Trigger

class CheckWorker(context: Context, params: WorkerParameters) : Worker(context, params) {
    override fun doWork(): Result {
        val t = runCatching { Trigger.valueOf(inputData.getString("trigger") ?: "TIMER") }.getOrDefault(Trigger.TIMER)
        if (inputData.getBoolean("logout", false)) {
            Checker.logout(applicationContext)
        } else {
            Checker.run(applicationContext, t)
        }
        return Result.success() // the engine does its own retry scheduling
    }

    companion object {
        fun input(t: Trigger): Data = triggerData(t)
    }
}
