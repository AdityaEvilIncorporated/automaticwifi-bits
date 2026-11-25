package io.github.autowifibits.app

import android.content.Context
import io.github.autowifibits.core.PortalConfig

/**
 * PortalConfig with the standard FortiGate defaults. Only the certificate pin can be changed in
 * the app (Advanced): it is the one setting the campus checklist may require on Android.
 */
class ConfigStore(context: Context) {
    private val p = context.getSharedPreferences("config", Context.MODE_PRIVATE)
    private val pinRe = Regex("^[0-9a-f]{64}$")

    fun load(): PortalConfig = PortalConfig(pinnedCertSha256 = pin)

    var pin: String
        get() = p.getString("pinned_cert_sha256", "")?.takeIf { pinRe.matches(it) } ?: ""
        set(v) {
            val clean = v.trim().lowercase()
            require(clean.isEmpty() || pinRe.matches(clean)) { "The pin must be 64 hex characters." }
            p.edit().putString("pinned_cert_sha256", clean).apply()
        }
}
