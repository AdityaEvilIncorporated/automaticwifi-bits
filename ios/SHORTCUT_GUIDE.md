# iOS Shortcut Guide (AutoWiFi BITS)

This guide shows you how to build an Apple Shortcut that logs in to the BITS campus Wi-Fi portal, and a Personal Automation that runs it automatically when your iPhone or iPad joins campus Wi-Fi.

## 1. What this does and its limits

- **Runs on Wi-Fi join only:** iOS does not allow background keepalive timers for Shortcuts, so there is no keepalive on iOS.
- **Session expiry:** If your session expires later, toggle Wi-Fi off and on, or run the Shortcut manually from the Shortcuts app.
- **Banners:** iOS Shortcuts may briefly show a banner when the automation runs.

## 2. Privacy

- Your BITS ID and password are stored in two **Text action**s inside the Shortcut on your device.
- If Shortcuts iCloud sync is enabled, the Shortcut (including those Text actions) syncs through **iCloud**. Turn on **Advanced Data Protection** in your iCloud settings for end-to-end encryption.
- **Never share** the Shortcut after you have filled in your real credentials. Use `YOUR_ID` and `YOUR_PASSWORD` only as placeholders while following this guide.

## 3. Build the Shortcut

Open the **Shortcuts** app, tap **+** to create a new Shortcut, and name it `AutoWiFi BITS`. Add these actions in order:

1. **Text** — enter `YOUR_ID` (your BITS ID).
2. **Text** — enter `YOUR_PASSWORD` (your portal password).
3. **Get Contents of URL** — set the URL to `http://neverssl.com` (Method: `GET`). This probes whether the campus firewall is intercepting your connection.
4. **Match Text** — set the pattern to:
   ```
   (?:fgtauth\?|name="magic" value=")([0-9A-Za-z]+)
   ```
   and the input to **Contents of URL** from step 3.
   - The magic token is a string matching `[0-9A-Za-z]+`.
   - **Why this pattern:** `Get Contents of URL` follows redirects. When the firewall replies with a 303 redirect to `fgtauth`, Shortcuts lands on the login form page, where the token appears in `<input type="hidden" name="magic" value="...">`. When the firewall replies with a 200 JavaScript page instead, `fgtauth?<magic>` appears directly in the text. This pattern handles both cases.
5. **If** *(required)* — set the condition to: **Matches** (from step 4) **does not have any value**.
   - Inside the **If** block, add **Stop This Shortcut**.
   - **End If**.
   - *Do not skip this step:* it ensures your credentials are never sent to the portal when you are already online or off campus.
6. **Get Group from Matched Text** — choose **Group At Index** `1` in **Matches** (from step 4). Its output, **Text from Group**, is the magic token used below. Do not put **Matches** itself into a URL or form field: it is a list holding the whole match (for example `fgtauth?abc123`), not the token.
7. **Get Contents of URL** — set the URL to `https://fw.bits-pilani.ac.in:8090/fgtauth?` followed by **Text from Group** (step 6).
   - This loads the login form and is only needed when the probe returned the 200 JS page; it is harmless otherwise.
8. **Get Contents of URL** — configure the login POST:
   - **URL:** `https://fw.bits-pilani.ac.in:8090/`
   - **Method:** `POST`
   - **Request Body:** `Form`
   - Add these four form fields:
     - `username` → **Text** from step 1
     - `password` → **Text** from step 2
     - `magic` → **Text from Group** from step 6
     - `4Tredir` → `http://neverssl.com/`
9. *(Optional)* **If** **Contents of URL** (from step 8) **contains** `authentication failed`:
   - **Show Notification** — `"AutoWiFi BITS: login failed, check your credentials."`
   - **End If**.

## 4. Set up the Personal Automation

1. Open the **Shortcuts** app and go to the **Automation** tab.
2. Tap **+** (New Automation) and choose **Wi-Fi**.
3. Tap **Choose** next to Network and select the campus **SSID** names:

| SSID | Status |
|---|---|
| *SSID names: to be confirmed on campus* | TODO |

4. Select **Run Immediately** (and turn **Notify When Run** off if you prefer).
5. Tap **Next**, choose your `AutoWiFi BITS` Shortcut, and tap **Done**.

## 5. Troubleshooting

- **Redirects and Match Text:** Because `Get Contents of URL` follows redirects, a 303 redirect from `http://neverssl.com` lands on the portal's login form page, where **Match Text** reads `name="magic" value="..."`. A 200 JS page matches `fgtauth?...` directly.
- **Different form field names:** If the portal does not accept the login, run `autowifi capture` on a laptop on campus to inspect the actual form field names, and update the `Form` fields in step 7.
- **Session expired:** iOS has no background keepalive. Toggle Wi-Fi off and on, or run the `AutoWiFi BITS` Shortcut manually.
