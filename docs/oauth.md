# OAuth Configuration & Architecture

This guide explains how to configure Google Cloud OAuth credentials for Gmail DeepClean, why specific scopes are requested, and how the application enforces strict read-only safety.

---

## 1. Google Cloud Console Setup

1. **Create or Select a Project**:
   - Go to the [Google Cloud Console](https://console.cloud.google.com/).
   - Create a new project (e.g., `Gmail DeepClean Local`) or select an existing one.

2. **Enable APIs**:
   - Navigate to **APIs & Services > Library**.
   - Search for **Gmail API** and click **Enable**.

3. **Configure OAuth Consent Screen**:
   - Navigate to **APIs & Services > OAuth consent screen**.
   - Choose **External** (or **Internal** if using Google Workspace).
   - Fill in the required application name and support email.
   - Add your own Gmail address under **Test users** (required while the app is in "Testing" mode).

4. **Create OAuth 2.0 Credentials**:
   - Navigate to **APIs & Services > Credentials**.
   - Click **Create Credentials > OAuth client ID**.
   - Select **Web application** as the application type.
   - Set **Authorized redirect URIs** to:
     ```
     http://localhost:8080/auth/gmail/callback
     ```
   - Click **Create** and copy your **Client ID** and **Client Secret**.

5. **Populate `.env`**:
   ```env
   GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your-client-secret
   REDIRECT_URL=http://localhost:8080/auth/gmail/callback
   PORT=8080
   HOST=127.0.0.1
   ```

---

## 2. OAuth Scopes & Rationale

Gmail DeepClean requests two OAuth scopes:

| Scope | URI | Purpose |
| :--- | :--- | :--- |
| **Full Mailbox Access** | `https://mail.google.com/` | Required by Google for IMAP XOAUTH2 authentication |
| **Gmail Read-Only** | `https://www.googleapis.com/auth/gmail.readonly` | Used for official Gmail REST API fallback |

### Why is `https://mail.google.com/` required?
To scan inboxes with thousands of messages in seconds, the application connects directly to Google's IMAP server (`imap.gmail.com:993`) using XOAUTH2. Google's IMAP service **rejects** restricted tokens like `gmail.readonly` and specifically requires the `mail.google.com` scope to authenticate.

---

## 3. Safety & Security Guarantees

Even though `https://mail.google.com/` permits modifications at the authorization level, Gmail DeepClean enforces strict read-only execution:

1. **Protocol-Level Protection (IMAP EXAMINE)**:
   - When connecting via IMAP, the application selects the mailbox using `c.Select(folder, true)`.
   - In IMAP, passing `true` issues the `EXAMINE` command instead of `SELECT`. The IMAP server explicitly locks the mailbox into a read-only state for that session, refusing any attempts to modify flags or alter messages.

2. **No Deletion or Sending Code**:
   - The application codebase contains zero functions or API calls that send, delete, move, or modify emails.
   - All email parsing extracts metadata (sender, recipient, subject, timestamp, size estimate) and writes it solely to your local SQLite database.

3. **Loopback Isolation**:
   - The HTTP server binds to `127.0.0.1` by default, preventing access from external networks or local LAN devices.
   - OAuth access and refresh tokens are stored securely in local SQLite (`data/deepclean.db`) and are never transmitted to third-party servers.
