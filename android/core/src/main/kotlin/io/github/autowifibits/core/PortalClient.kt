package io.github.autowifibits.core

import java.io.IOException

/** Result of [PortalClient.checkOnce]. Session fields only for LOGGED_IN; [error] only for PORTAL_ERROR. */
data class CheckResult(
    val outcome: Outcome,
    val magic: String = "",
    val keepaliveUrl: String = "",
    val logoutUrl: String = "",
    val error: String? = null,
)

/**
 * Network side of PROTOCOL.md section 2, a port of internal/portal/client.go (T22).
 * [probe] is used for the check URLs, [portal] for every portal request.
 */
class PortalClient(
    val cfg: PortalConfig,
    private val probe: Http,
    private val portal: Http,
    private val resolver: Resolver,
) : PortalApi {
    /** Step 1: an IP literal counts as resolved without a lookup. */
    fun onCampus(): Boolean =
        IPV4_REGEX.matches(cfg.portalHost) || resolver.resolves(cfg.portalHost)

    /** Steps 2-3: check_url, on IOException retry once with fallback_check_url; both fail → throws the last IOException. */
    fun probeOnce(): ProbeResult {
        val reply = try {
            probe.get(cfg.checkUrl)
        } catch (_: IOException) {
            probe.get(cfg.fallbackCheckUrl)
        }
        return Classify.classifyProbe(cfg, reply.status, reply.location, reply.body)
    }

    /** Steps 5-6; refuses (FOREIGN, no request) when !onPortal(magicUrl). [error] never contains credentials. */
    fun login(probe: ProbeResult, creds: Credentials): CheckResult {
        if (!Classify.onPortal(cfg, probe.magicUrl)) {
            return CheckResult(Outcome.FOREIGN)
        }

        val pageReply = try {
            portal.get(probe.magicUrl)
        } catch (e: IOException) {
            return CheckResult(Outcome.PORTAL_ERROR, error = safeError("login page GET failed", e))
        }
        if (pageReply.status !in 200..299) {
            return CheckResult(Outcome.PORTAL_ERROR, error = "login page HTTP ${pageReply.status}")
        }

        val form = try {
            Classify.parseLoginPage(cfg, pageReply.body, probe.magic)
        } catch (_: FormChangedException) {
            return CheckResult(Outcome.FORM_CHANGED)
        }

        val fields = listOf(
            cfg.usernameField to creds.username,
            cfg.passwordField to creds.password,
            cfg.magicField to form.magic,
            cfg.redirectField to form.redirect,
        )
        val postReply = try {
            portal.postForm(cfg.portalBase() + cfg.loginPath, fields)
        } catch (e: IOException) {
            return CheckResult(Outcome.PORTAL_ERROR, error = safeError("login POST failed", e))
        }

        val classified = Classify.classifyLoginReply(cfg, postReply.status, postReply.location, postReply.body)
        return when (classified.outcome) {
            Outcome.LOGGED_IN -> CheckResult(
                outcome = Outcome.LOGGED_IN,
                magic = form.magic,
                keepaliveUrl = classified.keepaliveUrl,
                logoutUrl = classified.logoutUrl,
            )
            Outcome.PORTAL_ERROR -> CheckResult(
                outcome = Outcome.PORTAL_ERROR,
                error = "login reply HTTP ${postReply.status}",
            )
            else -> CheckResult(outcome = classified.outcome)
        }
    }

    /** Whole check. [getCreds] is called only when captive; null → NO_CREDENTIALS. */
    override fun checkOnce(getCreds: () -> Credentials?): CheckResult {
        if (!onCampus()) {
            return CheckResult(Outcome.OFF_CAMPUS)
        }

        val probe = try {
            probeOnce()
        } catch (e: IOException) {
            return CheckResult(Outcome.PORTAL_ERROR, error = safeError("probe failed", e))
        }

        when (probe.kind) {
            ProbeKind.ONLINE -> return CheckResult(Outcome.ONLINE)
            ProbeKind.FOREIGN -> return CheckResult(Outcome.FOREIGN)
            ProbeKind.MALFORMED -> return CheckResult(Outcome.FORM_CHANGED)
            ProbeKind.CAPTIVE -> Unit
        }

        val creds = getCreds() ?: return CheckResult(Outcome.NO_CREDENTIALS)
        return login(probe, creds)
    }

    /** GET keepalive (only when onPortal) → Classify.keepaliveAlive; IOException → false. */
    override fun keepalive(url: String): Boolean {
        if (!Classify.onPortal(cfg, url)) return false
        return try {
            val reply = portal.get(url)
            Classify.keepaliveAlive(cfg, reply.status, reply.body)
        } catch (_: IOException) {
            false
        }
    }

    /** GET logout (only when onPortal), errors ignored. */
    override fun logout(url: String): Unit {
        if (!Classify.onPortal(cfg, url)) return
        try {
            portal.get(url)
        } catch (_: IOException) {
        }
    }

    private fun safeError(description: String, e: Throwable): String =
        "$description: ${e.javaClass.simpleName}"

    private companion object {
        val IPV4_REGEX = Regex("""^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$""")
    }
}
