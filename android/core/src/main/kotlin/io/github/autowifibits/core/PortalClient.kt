package io.github.autowifibits.core

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
    fun onCampus(): Boolean = TODO("T22")

    /** Steps 2-3: check_url, on IOException retry once with fallback_check_url; both fail → throws the last IOException. */
    fun probeOnce(): ProbeResult = TODO("T22")

    /** Steps 5-6; refuses (FOREIGN, no request) when !onPortal(magicUrl). [error] never contains credentials. */
    fun login(probe: ProbeResult, creds: Credentials): CheckResult = TODO("T22")

    /** Whole check. [getCreds] is called only when captive; null → NO_CREDENTIALS. */
    override fun checkOnce(getCreds: () -> Credentials?): CheckResult = TODO("T22")

    /** GET keepalive (only when onPortal) → Classify.keepaliveAlive; IOException → false. */
    override fun keepalive(url: String): Boolean = TODO("T22")

    /** GET logout (only when onPortal), errors ignored. */
    override fun logout(url: String): Unit = TODO("T22")
}
