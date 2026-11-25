package io.github.autowifibits.core

/** Check outcomes, PROTOCOL.md section 2. [wire] is the protocol name used in JSON cases. */
enum class Outcome(val wire: String) {
    OFF_CAMPUS("off_campus"), ONLINE("online"), FOREIGN("foreign"), LOGGED_IN("logged_in"),
    REJECTED("rejected"), FORM_CHANGED("form_changed"), PORTAL_ERROR("portal_error"),
    NO_CREDENTIALS("no_credentials");

    companion object {
        fun fromWire(s: String): Outcome = entries.first { it.wire == s }
    }
}

enum class ProbeKind(val wire: String) { ONLINE("online"), FOREIGN("foreign"), CAPTIVE("captive"), MALFORMED("malformed") }

/** magic and magicUrl are non-empty only for CAPTIVE; magicUrl = portalBase()+"/fgtauth?"+magic. */
data class ProbeResult(val kind: ProbeKind, val magic: String = "", val magicUrl: String = "")

data class LoginForm(val magic: String, val redirect: String, val action: String)

/** keepaliveUrl/logoutUrl non-empty only for LOGGED_IN. */
data class LoginReply(val outcome: Outcome, val keepaliveUrl: String = "", val logoutUrl: String = "")

/** Thrown by [parseLoginPage] when the page lacks the expected form. */
class FormChangedException : Exception("portal form changed")

/**
 * Pure classifiers, a line-by-line port of internal/portal/classify.go. Every function must pass
 * the matching file in protocol/testcases (SharedCasesTest).
 */
object Classify {
    /** PROTOCOL.md section 3. [location] may be null or "". */
    fun classifyProbe(cfg: PortalConfig, status: Int, location: String?, body: String): ProbeResult = TODO("T21")

    /** PROTOCOL.md section 4, HTML parsed with Jsoup. Throws [FormChangedException]. */
    fun parseLoginPage(cfg: PortalConfig, body: String, probeMagic: String): LoginForm = TODO("T21")

    /** PROTOCOL.md section 5 including the host rule. */
    fun classifyLoginReply(cfg: PortalConfig, status: Int, location: String?, body: String): LoginReply = TODO("T21")

    /** PROTOCOL.md section 7 "Alive". */
    fun keepaliveAlive(cfg: PortalConfig, status: Int, body: String): Boolean = TODO("T21")

    /** Host rule: https, host == portalHost (case-insensitive), port == portalPort (443 may be implicit). */
    fun onPortal(cfg: PortalConfig, url: String): Boolean = TODO("T21")
}
