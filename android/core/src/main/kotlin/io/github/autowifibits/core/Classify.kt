package io.github.autowifibits.core

import org.jsoup.Jsoup
import java.net.URI

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
    fun classifyProbe(cfg: PortalConfig, status: Int, location: String?, body: String): ProbeResult {
        if (status in 300..399 && !location.isNullOrEmpty()) {
            val uri = try {
                URI(location)
            } catch (_: Exception) {
                return ProbeResult(ProbeKind.FOREIGN)
            }
            if (!onPortal(cfg, location) || uri.path != "/fgtauth") {
                return ProbeResult(ProbeKind.FOREIGN)
            }
            val magic = uri.rawQuery ?: ""
            if (magic.isEmpty() || readAlnumToken(magic, 0) != magic) {
                return ProbeResult(ProbeKind.MALFORMED)
            }
            return ProbeResult(ProbeKind.CAPTIVE, magic = magic, magicUrl = "${cfg.portalBase()}/fgtauth?$magic")
        }

        val afterNeedle = findPortalUrlQueryStart(cfg, body, "fgtauth")
        if (afterNeedle >= 0) {
            val magic = readAlnumToken(body, afterNeedle)
            if (magic.isEmpty()) {
                return ProbeResult(ProbeKind.MALFORMED)
            }
            return ProbeResult(ProbeKind.CAPTIVE, magic = magic, magicUrl = "${cfg.portalBase()}/fgtauth?$magic")
        }

        return if (status == 200 || status == 204) {
            ProbeResult(ProbeKind.ONLINE)
        } else {
            ProbeResult(ProbeKind.FOREIGN)
        }
    }

    /** PROTOCOL.md section 4, HTML parsed with Jsoup. Throws [FormChangedException]. */
    fun parseLoginPage(cfg: PortalConfig, body: String, probeMagic: String): LoginForm {
        val doc = Jsoup.parse(body)
        for (form in doc.select("form")) {
            val inputs = form.select("input[name]")
            val byName = HashMap<String, String>(inputs.size)
            for (input in inputs) {
                val name = input.attr("name")
                if (name !in byName) {
                    byName[name] = input.attr("value")
                }
            }
            if (cfg.usernameField !in byName || cfg.passwordField !in byName) {
                continue
            }
            val formMagic = byName[cfg.magicField] ?: throw FormChangedException()
            val magic = formMagic.ifEmpty { probeMagic }
            val redirect = byName[cfg.redirectField] ?: ""
            return LoginForm(magic = magic, redirect = redirect, action = form.attr("action"))
        }
        throw FormChangedException()
    }

    /** PROTOCOL.md section 5 including the host rule. */
    fun classifyLoginReply(cfg: PortalConfig, status: Int, location: String?, body: String): LoginReply {
        val keepaliveToken = findPortalToken(cfg, location, body, "keepalive")
        if (keepaliveToken.isNotEmpty()) {
            val logoutToken = findPortalToken(cfg, location, body, "logout").ifEmpty { keepaliveToken }
            val base = cfg.portalBase()
            return LoginReply(
                outcome = Outcome.LOGGED_IN,
                keepaliveUrl = "$base/keepalive?$keepaliveToken",
                logoutUrl = "$base/logout?$logoutToken",
            )
        }

        if (body.contains("authentication failed", ignoreCase = true) || hasPasswordInput(cfg, body)) {
            return LoginReply(Outcome.REJECTED)
        }
        if (status !in 200..399) {
            return LoginReply(Outcome.PORTAL_ERROR)
        }
        return LoginReply(Outcome.FORM_CHANGED)
    }

    /** PROTOCOL.md section 7 "Alive". */
    fun keepaliveAlive(cfg: PortalConfig, status: Int, body: String): Boolean {
        if (status !in 200..299) {
            return false
        }
        if (body.contains("fgtauth", ignoreCase = true)) {
            return false
        }
        return !hasPasswordInput(cfg, body)
    }

    /** Host rule: https, host == portalHost (case-insensitive), port == portalPort (443 may be implicit). */
    fun onPortal(cfg: PortalConfig, url: String): Boolean {
        if (url.isBlank()) return false
        return try {
            val uri = URI(url)
            val scheme = uri.scheme ?: return false
            val host = uri.host ?: return false
            val port = if (uri.port == -1) 443 else uri.port
            scheme.equals("https", ignoreCase = true) &&
                host.equals(cfg.portalHost, ignoreCase = true) &&
                port == cfg.portalPort
        } catch (_: Exception) {
            false
        }
    }

    private fun hasPasswordInput(cfg: PortalConfig, body: String): Boolean =
        Jsoup.parse(body).select("input[name]").any { it.attr("name") == cfg.passwordField }

    /** Earliest portal URL `{base}/{path}?<token>` with a non-empty token: the Location header
     *  first, then the body. Token-less mentions (e.g. a JS template) are skipped, as in Go. */
    private fun findPortalToken(cfg: PortalConfig, location: String?, body: String, path: String): String {
        if (!location.isNullOrEmpty()) {
            val tok = firstToken(cfg, location, path)
            if (tok.isNotEmpty()) return tok
        }
        return firstToken(cfg, body, path)
    }

    private fun firstToken(cfg: PortalConfig, text: String, path: String): String {
        val host = cfg.portalHost.lowercase()
        val port = cfg.portalPort
        var bestIdx = -1
        var best = ""
        for (needle in listOf("https://$host:$port/$path?", "https:\\/\\/$host:$port\\/$path?")) {
            var from = 0
            while (true) {
                val idx = text.indexOf(needle, from, ignoreCase = true)
                if (idx < 0) break
                val tok = readAlnumToken(text, idx + needle.length)
                if (tok.isNotEmpty()) {
                    if (bestIdx < 0 || idx < bestIdx) { bestIdx = idx; best = tok }
                    break
                }
                from = idx + needle.length
            }
        }
        return best
    }

    /** Index just after the first `{base}/{path}?` (plain or JSON-escaped), or -1 (probe only). */
    private fun findPortalUrlQueryStart(cfg: PortalConfig, text: String, path: String): Int {
        val host = cfg.portalHost.lowercase()
        val port = cfg.portalPort
        val plain = "https://$host:$port/$path?"
        val escaped = "https:\\/\\/$host:$port\\/$path?"
        val i1 = text.indexOf(plain, ignoreCase = true)
        val i2 = text.indexOf(escaped, ignoreCase = true)
        return when {
            i1 >= 0 && (i2 < 0 || i1 <= i2) -> i1 + plain.length
            i2 >= 0 -> i2 + escaped.length
            else -> -1
        }
    }

    private fun readAlnumToken(text: String, start: Int): String {
        var i = start
        while (i < text.length) {
            val c = text[i]
            if (c in '0'..'9' || c in 'A'..'Z' || c in 'a'..'z') {
                i++
            } else {
                break
            }
        }
        return text.substring(start, i)
    }
}
