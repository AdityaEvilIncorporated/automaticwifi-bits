# AutoWiFi BITS protocol (v1)

One spec for every client (Go desktop, Kotlin Android, iOS Shortcut). The shared cases in
`protocol/testcases/*.json` are normative: both the Go and the Kotlin code run them. When
this text and a case disagree, the case wins and this text gets fixed.

All names in `monospace` are configuration keys (see "Configuration"); defaults in brackets.

## 1. Configuration

| key | default | meaning |
|---|---|---|
| `check_url` | `http://neverssl.com` | plain-HTTP probe |
| `fallback_check_url` | `http://connectivitycheck.gstatic.com/generate_204` | probe used when `check_url` has a network error |
| `portal_host` | `fw.bits-pilani.ac.in` | captive portal host |
| `portal_port` | `8090` | captive portal port (always HTTPS) |
| `login_path` | `/` | path the login form is POSTed to |
| `username_field` | `username` | form field names |
| `password_field` | `password` | |
| `magic_field` | `magic` | |
| `redirect_field` | `4Tredir` | |
| `pinned_cert_sha256` | empty | lowercase hex SHA-256 of the portal leaf certificate's SubjectPublicKeyInfo; empty = no pin |

"Portal base" = `https://{portal_host}:{portal_port}`. Port 443 is still written explicitly.

## 2. One check (`check_once`)

A check returns exactly one **outcome**:
`off_campus`, `online`, `foreign`, `logged_in`, `rejected`, `form_changed`, `portal_error`,
`no_credentials`.

1. **Campus test.** Resolve `portal_host` with DNS (an IP literal counts as resolved).
   Failure → `off_campus`. Nothing else happens; no notification.
2. **Probe.** `GET check_url`, redirects NOT followed, timeout 10 s, read at most 256 KiB.
   A network error (connect, timeout, TLS) → retry once with `fallback_check_url`.
   Both fail → `portal_error`.
3. **Classify the probe** (section 3) → `online` | `foreign` | `captive(magic_url)` | `malformed`.
   `online` and `foreign` end the check. `malformed` → `form_changed`.
4. **Credentials.** None stored → `no_credentials`.
5. **Login page.** `GET magic_url` on the portal (HTTPS, TLS rules in section 6), redirects not
   followed. Network error or non-2xx → `portal_error`. Parse it (section 4); a missing
   expected field → `form_changed`.
6. **Login.** `POST {portal base}{login_path}`, `Content-Type: application/x-www-form-urlencoded`,
   fields (in this order): `username_field`=username, `password_field`=password,
   `magic_field`=magic, `redirect_field`=redirect value. Classify the reply (section 5).
   Network error → `portal_error`.

**Host rule (security):** credentials are only ever sent to the portal base. A `magic_url`,
keepalive URL or logout URL whose scheme is not `https`, or whose host or port differ from
`portal_host`/`portal_port`, is never fetched. For a magic URL this classifies as `foreign`; for
keepalive/logout URLs found in a login reply, they are dropped (the login still counts as
`logged_in` when a valid keepalive URL exists; with none valid it is `form_changed`).

## 3. Classifying the probe reply

Input: HTTP status, `Location` header (may be empty), body (bytes, decoded as UTF-8, invalid
bytes replaced). "Portal fgtauth URL" = a URL whose scheme is `https`, host equals
`portal_host` (case-insensitive), port equals `portal_port`, path is `/fgtauth`.

1. Status 3xx with a `Location`:
   - Location is a portal fgtauth URL whose raw query string is one or more magic characters
     `[0-9A-Za-z]` → `captive`, magic = that query string.
   - Location is a portal fgtauth URL with an empty query, or a query with any other character
     → `malformed`.
   - Anything else → `foreign` (another network's captive portal, or a CDN redirect).
2. Otherwise search the body for the first occurrence of the literal text
   `https://{portal_host}:{portal_port}/fgtauth?` (host matched case-insensitively; the forms
   `https:\/\/host:port\/fgtauth?` used inside JavaScript strings also count).
   - Found and followed by one or more magic characters `[0-9A-Za-z]` → `captive`, magic = that
     run of characters.
   - Found with no magic characters after it → `malformed`.
3. Otherwise: status 200 or 204 → `online`. Any other status → `foreign`.

`magic_url` is always rebuilt as `{portal base}/fgtauth?{magic}`, never taken verbatim.

## 4. Parsing the login page

Parse HTML forms (tolerant parser; attribute names case-insensitive). Pick the first `<form>`
that contains an `<input>` named `username_field` AND one named `password_field`.
- No such form → `form_changed`.
- The form must also contain an input named `magic_field`; missing → `form_changed`.
  If its `value` is non-empty it replaces the magic from the probe.
- `redirect_field` is optional; its `value` (or empty string) is sent back.
- The form `action` is reported by `capture` but NOT used for the POST: the POST always goes to
  `{portal base}{login_path}` (host rule).

## 5. Classifying the login reply

Input: status, Location, body.
1. Collect every portal URL in the body or Location of the form
   `{portal base}/keepalive?{token}` and `{portal base}/logout?{token}`, token `[0-9A-Za-z]+`
   (also the `\/`-escaped forms). First of each wins.
2. A keepalive URL found → `logged_in`, with `keepalive_url`, and `logout_url` = the logout URL
   found, or else `{portal base}/logout?{keepalive token}`.
3. Else body contains `authentication failed` (case-insensitive) OR still contains an input named
   `password_field` → `rejected`.
4. Else status is not 2xx/3xx → `portal_error`.
5. Else → `form_changed` (the portal answered something we do not understand).

## 6. TLS

The probe is plain HTTP by design. Every portal request is HTTPS and verified:
1. Verify the chain against the OS trust store with hostname `portal_host`. Pass → use it.
2. Fail, and `pinned_cert_sha256` is set, and SHA-256(leaf SubjectPublicKeyInfo) equals it →
   accept for this host only.
3. Otherwise refuse (`portal_error`). Verification is never disabled; the pin applies to the
   portal client only.

## 7. Keepalive and logout

- After `logged_in`, `GET keepalive_url` every 5 minutes.
- Alive = status 2xx and the body contains neither `fgtauth` nor an input named `password_field`.
  Anything else (including 3xx and network errors) = expired: drop the session and run a
  fresh check immediately.
- Logout (manual, menu item) = `GET logout_url`, then drop the session. Errors are ignored.

## 8. Scheduling, retries, notifications

- Triggers: start, network change, periodic timer (5 min desktop; 15 min WorkManager on Android),
  manual "Log in now", credentials updated.
- Retry delays after a failed check (`portal_error`, `form_changed`): 5 s, 15 s, 60 s, then every
  5 min. Any `online`, `logged_in`, `off_campus` or `foreign` resets the failure counter.
- `rejected`: set `credentials_rejected`; no automatic retries until the credentials are updated
  (or the user presses "Log in now", which tries once).
- Notifications (quiet, at most one per hour across all reasons, never on success):
  `rejected`; the 3rd consecutive `portal_error`; `form_changed`; `no_credentials` when the portal
  is showing.
- Texts never contain the username, password, magic or tokens.

## 9. Logging

Logs may contain outcomes, status codes, hosts and durations. Logs never contain the password,
username, magic, keepalive/logout tokens or full portal URLs with query strings.

## 10. Shared test cases

`protocol/testcases/`:
- `probe.json`: `{config?, status, location, body} -> {kind, magic}`; kind is `online`,
  `foreign`, `captive`, `malformed`.
- `login_page.json`: `{body, probe_magic} -> {ok, magic, redirect, error}`; error is
  `form_changed` or null.
- `login_reply.json`: `{status, location, body} -> {outcome, keepalive_url, logout_url}`.
- `keepalive.json`: `{status, body} -> {alive}`.
- `scenarios.json`: end-to-end cases against the mock portal: mock settings, the stored
  credentials, and the expected outcome of `check_once` (plus counters).

Cases use the default config unless they carry a `config` object that overrides keys.

## 11. Mock portal (`cmd/mockportal`)

A local FortiGate stand-in used by every test. Plain-HTTP probe server + HTTPS portal server
with its own test CA. Per-visit magic tokens (a magic works for one POST), keepalive and logout
endpoints, and a control API under `/__mock/` on the probe server: `reset`, `expire`,
`fail?n=N` (next N portal requests answer 500), `fields?username=..&password=..&magic=..`,
`style?probe=redirect|page|js`, `state` (JSON counters). See `internal/mockportal/doc.go`.
