package io.github.autowifibits.core

/** Portal configuration, PROTOCOL.md section 1. Defaults are the standard FortiGate values. */
data class PortalConfig(
    val checkUrl: String = "http://neverssl.com",
    val fallbackCheckUrl: String = "http://connectivitycheck.gstatic.com/generate_204",
    val portalHost: String = "fw.bits-pilani.ac.in",
    val portalPort: Int = 8090,
    val loginPath: String = "/",
    val usernameField: String = "username",
    val passwordField: String = "password",
    val magicField: String = "magic",
    val redirectField: String = "4Tredir",
    /** Lowercase hex SHA-256 of the portal leaf SubjectPublicKeyInfo, or "". */
    val pinnedCertSha256: String = "",
) {
    /** "https://{host lower-cased}:{port}", port always written. */
    fun portalBase(): String = "https://${portalHost.lowercase()}:$portalPort"

    companion object {
        /**
         * Applies the snake_case keys of a shared-test-case `config` object (PROTOCOL.md keys:
         * check_url, portal_host, portal_port, username_field, ...) on top of the defaults.
         * Unknown keys are ignored.
         */
        fun fromOverrides(overrides: Map<String, Any?>): PortalConfig {
            val d = PortalConfig()
            return PortalConfig(
                checkUrl = overrides["check_url"] as? String ?: d.checkUrl,
                fallbackCheckUrl = overrides["fallback_check_url"] as? String ?: d.fallbackCheckUrl,
                portalHost = overrides["portal_host"] as? String ?: d.portalHost,
                portalPort = (overrides["portal_port"] as? Number)?.toInt() ?: d.portalPort,
                loginPath = overrides["login_path"] as? String ?: d.loginPath,
                usernameField = overrides["username_field"] as? String ?: d.usernameField,
                passwordField = overrides["password_field"] as? String ?: d.passwordField,
                magicField = overrides["magic_field"] as? String ?: d.magicField,
                redirectField = overrides["redirect_field"] as? String ?: d.redirectField,
                pinnedCertSha256 = overrides["pinned_cert_sha256"] as? String ?: d.pinnedCertSha256,
            )
        }
    }
}

/** Campus credentials. toString() never shows the password. */
data class Credentials(val username: String, val password: String) {
    /** "Credentials(username=<u>, password=[redacted])"; "password=[]" when empty. */
    override fun toString(): String {
        val pw = if (password.isEmpty()) "[]" else "[redacted]"
        return "Credentials(username=$username, password=$pw)"
    }
}
