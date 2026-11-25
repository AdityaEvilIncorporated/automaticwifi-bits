package io.github.autowifibits.app

import android.Manifest
import android.annotation.SuppressLint
import android.app.Activity
import android.content.Intent
import android.content.SharedPreferences
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.PowerManager
import android.provider.Settings
import android.text.InputType
import android.view.View
import android.view.ViewGroup.LayoutParams.MATCH_PARENT
import android.view.ViewGroup.LayoutParams.WRAP_CONTENT
import android.view.WindowInsets
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.workDataOf
import io.github.autowifibits.core.Credentials
import io.github.autowifibits.core.Trigger

/** The only screen: ID, password, status, "Log in now", "Log out", and the pin under Advanced. */
class MainActivity : Activity() {
    private lateinit var status: TextView
    private val stateListener = SharedPreferences.OnSharedPreferenceChangeListener { _, _ -> runOnUiThread { refresh() } }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Checker.ensureChannel(this)
        Scheduler.ensure(this)
        val store = CredentialStore(this)
        val config = ConfigStore(this)
        val pad = (16 * resources.displayMetrics.density).toInt()
        val col = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL; setPadding(pad, pad, pad, pad) }
        fun add(v: View) = col.addView(v, LinearLayout.LayoutParams(MATCH_PARENT, WRAP_CONTENT).apply { topMargin = pad / 2 })

        add(TextView(this).apply { text = "AutoWiFi BITS"; textSize = 22f })
        add(TextView(this).apply {
            text = "Logs you in to BITS campus Wi-Fi automatically. Your ID and password are encrypted " +
                "with a key in this phone's Keystore and are only sent to https://fw.bits-pilani.ac.in:8090."
        })
        status = TextView(this).also { add(it) }
        val id = EditText(this).apply { hint = "BITS campus ID"; inputType = InputType.TYPE_CLASS_TEXT; setText(store.get()?.username ?: "") }
        val pw = EditText(this).apply { hint = "Password"; inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD }
        add(id); add(pw)
        add(Button(this).apply {
            text = "Save"
            setOnClickListener {
                val c = Credentials(id.text.toString().trim(), pw.text.toString())
                try {
                    store.set(c)
                } catch (e: IllegalArgumentException) {
                    status.text = "Enter both ID and password."; return@setOnClickListener
                } catch (e: Exception) {
                    status.text = "Could not save in the Keystore (${e.javaClass.simpleName})."; return@setOnClickListener
                }
                pw.setText("")
                status.text = "Saved. Checking…"
                Scheduler.now(this@MainActivity, Trigger.CREDS_UPDATED)
                askBatteryOnce()
            }
        })
        add(Button(this).apply { text = "Log in now"; setOnClickListener { Scheduler.now(this@MainActivity, Trigger.MANUAL); status.text = "Checking…" } })
        add(Button(this).apply {
            text = "Log out"
            setOnClickListener {
                WorkManager.getInstance(this@MainActivity).enqueue(
                    OneTimeWorkRequestBuilder<CheckWorker>().setInputData(workDataOf("logout" to true)).build())
                status.text = "Logging out…"
            }
        })

        // Advanced: only needed if the campus checklist says Android does not trust the portal.
        val adv = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL; visibility = View.GONE }
        val pin = EditText(this).apply { hint = "pinned_cert_sha256 (64 hex, from 'autowifi capture')"; setText(config.pin) }
        val pinMsg = TextView(this)
        adv.addView(pin); adv.addView(pinMsg)
        adv.addView(Button(this).apply {
            text = "Save pin"
            setOnClickListener {
                pinMsg.text = try { config.pin = pin.text.toString(); "Saved." } catch (e: IllegalArgumentException) { e.message }
            }
        })
        add(Button(this).apply { text = "Advanced"; setOnClickListener { adv.visibility = if (adv.visibility == View.GONE) View.VISIBLE else View.GONE } })
        add(adv)

        val scroll = ScrollView(this).apply { addView(col) }
        // targetSdk 35 is edge-to-edge on Android 15: keep content clear of the system bars.
        scroll.setOnApplyWindowInsetsListener { v, insets ->
            val (top, bottom) = if (Build.VERSION.SDK_INT >= 30) {
                val b = insets.getInsets(WindowInsets.Type.systemBars() or WindowInsets.Type.displayCutout())
                b.top to b.bottom
            } else {
                @Suppress("DEPRECATION")
                insets.systemWindowInsetTop to insets.systemWindowInsetBottom
            }
            v.setPadding(0, top, 0, bottom)
            insets
        }
        setContentView(scroll)

        if (Build.VERSION.SDK_INT >= 33 && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 1)
        }
    }

    override fun onResume() {
        super.onResume()
        getSharedPreferences("state", MODE_PRIVATE).registerOnSharedPreferenceChangeListener(stateListener)
        refresh()
    }

    override fun onPause() {
        getSharedPreferences("state", MODE_PRIVATE).unregisterOnSharedPreferenceChangeListener(stateListener)
        super.onPause()
    }

    /** Status from the saved state; called on resume and whenever a check writes new state. */
    private fun refresh() {
        val st = StateStore(this)
        val last = st.lastOutcome
        status.text = when {
            CredentialStore(this).get() == null -> "Enter your campus ID and password."
            st.load().credentialsRejected -> "The portal rejected your ID or password. Save them again."
            last.isEmpty() -> "Waiting for the first check."
            else -> "Last check: " + Logic.label(last)
        }
    }

    /** Ask once for the battery-optimisation exemption; the app keeps working if refused. */
    @SuppressLint("BatteryLife") // sideloaded APK; the spec asks for this prompt once
    private fun askBatteryOnce() {
        val prefs = getSharedPreferences("ui", MODE_PRIVATE)
        if (prefs.getBoolean("asked_battery", false)) return
        prefs.edit().putBoolean("asked_battery", true).apply()
        val pm = getSystemService(PowerManager::class.java)
        if (pm.isIgnoringBatteryOptimizations(packageName)) return
        runCatching {
            startActivity(Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS, Uri.parse("package:$packageName")))
        }
    }
}
