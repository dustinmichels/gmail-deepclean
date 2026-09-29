# Google OAuth 2.0 Configuration & Architecture

This document describes how Google OAuth 2.0 is configured, implemented, and managed in Gmail DeepClean.

---

## 1. Quick Overview

- **Protocol**: OAuth 2.0 Authorization Code flow with PKCE-equivalent cryptographically secure CSRF protection.
- **Access Scope**: `https://www.googleapis.com/auth/gmail.readonly` (**Strictly read-only**). The app cannot send, modify, or delete emails.
- **Offline Access**: Enabled (`AccessTypeOffline`). The app requests a refresh token so user sessions remain active without repeated re-consent.
- **Session Architecture**: OAuth access and refresh tokens are stored securely in the local SQLite database (`data/deepclean.db`). The client receives a random 32-byte session token passed as `Authorization: Bearer <sessionToken>`.

---

## 2. Why Google Cloud Console Shows "This OAuth client has not been used"

If you just authenticated successfully but Google Cloud Console displays:

```text
OAuth 2.0 Client IDs
- DeepClean-local
- Gmail Deepclean
[!] This OAuth client has not been used
```

This occurs for two reasons:

### A. Google Cloud Metric Aggregation Latency (6–48 hours)
Google Cloud Console's Credentials table does **not** update in real time. Usage metrics (such as "Last used date" and traffic counters) are processed by Google's asynchronous Cloud Monitoring and telemetry pipelines. It typically takes between **6 to 48 hours** for token exchanges and API requests to register in the Console UI. 

> **Proof that it is working**: If Google had rejected the Client ID or Secret, the authorization page or token exchange would have immediately failed with `400 redirect_uri_mismatch` or `401 invalid_client`. Because authentication succeeded and your email/messages loaded, Google accepted the credentials.

### B. Multiple Client IDs in the Same Project
You have two client IDs created in Google Cloud Console (`DeepClean-local` and `Gmail Deepclean`). The application uses only **one** client ID at a time—the one configured in your `.env` file under `GOOGLE_CLIENT_ID`.
- To identify which client ID is being used:
  1. Open `.env` and inspect the `GOOGLE_CLIENT_ID` value (starts with a project number, e.g., `1007013360109-...apps.googleusercontent.com`).
  2. In Google Cloud Console under **APIs & Services > Credentials**, click on `DeepClean-local` and `Gmail Deepclean`.
  3. Compare the **Client ID** string to see which matches `.env`.
- The matching client ID is the active one (metrics will show up after the reporting delay). The other client ID has never been used by the app and will remain flagged as unused unless you switch to it.

---

## 3. Configuration & Environment Variables

The application reads configuration at startup via `api.Init()` in `api/config.go` from environment variables (loaded from `.env` by `godotenv` in `main.go`).

### Environment Variables

| Variable | Description | Example / Default |
| :--- | :--- | :--- |
| `GOOGLE_CLIENT_ID` | OAuth 2.0 Client ID generated in GCP | `1007013...apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | OAuth 2.0 Client Secret generated in GCP | `GOCSPX-...` |
| `REDIRECT_URL` | Exact redirect URI registered in GCP | `http://localhost:8080/auth/gmail/callback` |
| `PORT` | Local HTTP server port | `8080` |
| `DB_PATH` | Path to persistent SQLite database | `data/deepclean.db` |

Template file: `.env.example`.

---

## 4. Google Cloud Console Setup Guide

To configure a new or existing Google Cloud project for Gmail DeepClean:

### Step 1: Enable the Gmail API
1. Navigate to [Google Cloud Console](https://console.cloud.google.com/).
2. Select your project.
3. Go to **APIs & Services > Library**.
4. Search for **Gmail API** and click **Enable**.

### Step 2: Configure the OAuth Consent Screen
1. Go to **APIs & Services > OAuth consent screen**.
2. Select **User Type**:
   - Choose **External** (unless you are using Google Workspace internal accounts).
3. Fill in required App Information:
   - **App name**: `Gmail DeepClean`
   - **User support email**: Your email address
   - **Developer contact information**: Your email address
4. Click **Save and Continue** to go to **Scopes**:
   - Click **Add or Remove Scopes**.
   - Select `.../auth/gmail.readonly` (`Gmail API: View your mail messages and settings`).
   - Click **Update** then **Save and Continue**.
5. Add **Test Users**:
   - While the app publishing status is **Testing**, Google restricts login to registered test users.
   - Add your Gmail address to the test users list.
   - Click **Save and Continue**.

### Step 3: Create OAuth 2.0 Client ID
1. Go to **APIs & Services > Credentials**.
2. Click **Create Credentials > OAuth client ID**.
3. Set **Application type**: **Web application** (do not choose Desktop app).
4. Set **Name**: E.g., `DeepClean-local`.
5. Under **Authorized JavaScript origins**:
   - Add: `http://localhost:8080`
   - Add: `http://127.0.0.1:8080` (optional)
6. Under **Authorized redirect URIs**:
   - Add: `http://localhost:8080/auth/gmail/callback`
   *(Must match `REDIRECT_URL` in `.env` character-for-character, including scheme and port)*.
7. Click **Create**.
8. Copy the **Client ID** and **Client Secret** into your `.env` file.

---

## 5. End-to-End OAuth & Session Architecture

```
User (Browser)               Go Backend Server             Google OAuth / API             SQLite DB
      |                              |                              |                        |
      |--- 1. Click "Connect" ------>|                              |                        |
      |    (window.open)             |                              |                        |
      |    GET /auth/gmail           |                              |                        |
      |                              |-- 2. Generate 32-byte CSRF ->|                        |
      |                              |      state token             |                        |
      |                              |------------------------------|------ Save CSRF ------>|
      |                              |<-----------------------------|------ (15m TTL) -------|
      |<-- 3. 307 Redirect to -------|                              |                        |
      |       Google OAuth URL       |                              |                        |
      |       (state, offline, force)|                              |                        |
      |                              |                              |                        |
      |--- 4. User Consents --------------------------------------->|                        |
      |                              |                              |                        |
      |<-- 5. Redirect with Code -----------------------------------|                        |
      |       GET /auth/gmail/callback?code=...&state=...           |                        |
      |                              |                              |                        |
      |----------------------------->|                              |                        |
      |                              |-- 6. Validate CSRF state ---------------------------->|
      |                              |      (single-use consume)    |<----- Validated -------|
      |                              |                              |                        |
      |                              |-- 7. Exchange code --------->|                        |
      |                              |      (client_id, secret)     |                        |
      |                              |<-- 8. Return tokens ---------|                        |
      |                              |       (access, refresh)      |                        |
      |                              |                              |                        |
      |                              |-- 9. Users.GetProfile("me") >|                        |
      |                              |<-- Return user email --------|                        |
      |                              |                              |                        |
      |                              |-- 10. Store session & tokens ------------------------>|
      |                              |       (30-day sessionToken)  |<----- Stored ----------|
      |                              |                              |                        |
      |<-- 11. HTML + postMessage ---|                              |                        |
      |    { type: 'GMAIL_AUTH_SUCCESS', sessionToken: ... }        |                        |
      |                              |                              |                        |
      |-- 12. API Requests --------->|                              |                        |
      |   Authorization: Bearer <tok>|-- 13. Auto-refresh if needed >|                        |
      |                              |<-- Return Gmail data --------|                        |
```

### Flow Step-by-Step

1. **Initiation (`GET /auth/gmail` in `api/auth.go`)**:
   - The user clicks "Connect Gmail" in the frontend.
   - The frontend opens a popup window pointing to `/auth/gmail`.
   - The server creates a cryptographically secure random 32-byte token via `crypto/rand`.
   - The state is persisted in SQLite with a 15-minute TTL (`DB.SaveCSRFState`) and set in an HTTP-only `SameSite=Lax` cookie `gmail_oauth_csrf` for defense-in-depth.
   - The server redirects the user to Google's OAuth 2.0 authorization endpoint:
     ```go
     authURL := oauthConfig.AuthCodeURL(
         state,
         oauth2.AccessTypeOffline, // Request refresh token
         oauth2.ApprovalForce,     // Force consent screen to guarantee refresh token delivery
     )
     ```

2. **User Consent**:
   - The user signs in and grants read-only Gmail access.
   - Google redirects the browser back to `http://localhost:8080/auth/gmail/callback?code=...&state=...`.

3. **Callback & Token Exchange (`GET /auth/gmail/callback` in `api/auth.go`)**:
   - The server verifies that the received `state` exists in the database and is still valid. Once verified, the state is immediately deleted from SQLite (single-use token) to prevent replay attacks.
   - The server exchanges the authorization code for an OAuth2 token using Google's token endpoint (`oauthConfig.Exchange`).
   - The server calls the Gmail API `Users.GetProfile("me")` to obtain the authenticated email address.
   - The user is upserted in the SQLite `users` table.

4. **Session Creation**:
   - The server generates a random 32-byte session token.
   - The OAuth token (access token, refresh token, expiry) is saved in the SQLite `sessions` table linked to the session token with a 30-day lifetime.
   - Raw Google OAuth tokens are **never** transmitted directly to client JavaScript.
   - The callback responds with an HTML page that communicates with the parent window using `window.opener.postMessage(...)` and closes itself.

5. **API Authentication & Transparent Refresh (`api/auth.go`)**:
   - Every protected API endpoint calls `api.ParseSession(r)`.
   - It extracts the session token from `Authorization: Bearer <sessionToken>`.
   - It loads the session and stored OAuth token from SQLite.
   - If the OAuth access token is expired or within 2 minutes of expiration, it automatically uses the refresh token to request a new access token from Google:
     ```go
     if sess.OAuthToken != nil && sess.OAuthToken.Expiry.Before(time.Now().Add(2*time.Minute)) {
         tokenSource := oauthConfig.TokenSource(r.Context(), sess.OAuthToken)
         newToken, err := tokenSource.Token()
         if err == nil && newToken.AccessToken != sess.OAuthToken.AccessToken {
             sess.OAuthToken = newToken
             _ = DB.UpdateSessionOAuthToken(sess.Token, newToken)
         }
     }
     ```
   - If valid, the request proceeds.

6. **Logout (`POST /api/auth/logout`)**:
   - Invalidates and removes the session from the SQLite database.

---

## 6. Security Guarantees

1. **Read-Only Scope**: The application only requests `https://www.googleapis.com/auth/gmail.readonly`. It is technically incapable of sending, modifying, trashing, or deleting messages through Google's API.
2. **Server-Side Token Storage**: OAuth access and refresh tokens stay in the local SQLite database on the backend server. The browser only holds a high-entropy session token.
3. **CSRF Mitigation**: Dual-layer protection using server-persisted single-use state tokens with a strict 15-minute TTL plus HTTP-only `SameSite=Lax` cookies.
4. **Origin-Bound Messaging**: The OAuth popup communication uses explicit origin checks (`window.location.origin`) to prevent cross-origin message interception.

---

## 7. Troubleshooting

| Symptom | Cause | Solution |
| :--- | :--- | :--- |
| `redirect_uri_mismatch` (400) | `REDIRECT_URL` in `.env` does not match the URI configured in Google Cloud Console. | Ensure `http://localhost:8080/auth/gmail/callback` is listed under **Authorized redirect URIs** in GCP without trailing slashes. |
| `Access blocked: App has not completed the Google verification process` (403) | Project is in **Testing** mode and the logged-in email is not on the Test Users list. | In GCP Console, go to **OAuth consent screen > Test users** and add your Google account. |
| `invalid_client` (401) | Incorrect `GOOGLE_CLIENT_ID` or `GOOGLE_CLIENT_SECRET`. | Double-check that `.env` contains the exact credentials copied from GCP Console. |
| Popup closes but main window doesn't update | Browser popup blocker or cross-origin restriction. | Allow popups for `localhost:8080` and refresh the dashboard. |
| "This OAuth client has not been used" in GCP | Telemetry reporting latency (up to 48h) or looking at the inactive client ID. | Verify `.env` against the Client ID in GCP; give Google Cloud metrics time to aggregate. |
