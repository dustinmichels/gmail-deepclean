# Gmail DeepClean

A local-first tool for analyzing and cleaning your Gmail inbox using high-performance IMAP envelope streaming with Gmail REST API fallback, strictly executed in **read-only** mode.

## Features

- **Strictly Read-Only Execution**: The app never modifies, deletes, or sends emails. Folders are opened in read-only mode via IMAP `EXAMINE`, and all application logic is strictly analytical.
- **High-Performance Scanning**: Rapidly indexes inbox metadata via IMAP XOAUTH2 streaming with automatic fallback to the official Gmail REST API.
- **Contacts Are Off-Limits**: Anyone you have ever sent an email to is treated as a real contact. Their mail is never shown. The dashboard, sender lists, and email lists only cover mail from people you have never emailed.
- **Inbox Analytics**: Totals count every scanned email. The dashboard shows what percent of all emails and all mailbox space comes from people you have never emailed, and ranks those senders by volume and size.
- **Local & Private**: All data and OAuth tokens are stored in a local SQLite database (`data/deepclean.db`). Binds to `127.0.0.1` by default.
- **Secure Session Management**: Server-side token storage, automated token refresh, and CSRF protection.

---

## Quickstart

### 1. Prerequisites

- [Go](https://go.dev/) (1.23+)
- [Bun](https://bun.sh/) (or Node.js)
- Google Cloud Project with the Gmail API enabled

### 2. Configure Environment

Copy the template environment file:

```bash
cp .env.example .env
```

Edit `.env` and supply your Google Cloud OAuth credentials:

```env
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
REDIRECT_URL=http://localhost:8080/auth/gmail/callback
PORT=8080
```


### OAuth Permissions & Security

To support high-speed mailbox scanning via IMAP (`imap.gmail.com`), Google requires the `https://mail.google.com/` scope for XOAUTH2 authentication (Google does not support `gmail.readonly` over IMAP).

To guarantee safety:
1. **Protocol-Level Read-Only**: The IMAP client opens mailboxes with the IMAP `EXAMINE` command (`readOnly: true`), ensuring the server rejects any attempt to modify or delete messages or flags.
2. **Zero Modification Code**: The codebase contains strictly zero code paths for deleting, altering, moving, or sending messages.
3. **Local-First Isolation**: The server binds to local loopback (`127.0.0.1`) by default, and tokens remain exclusively in your local SQLite database.

For detailed setup instructions and scope details, see [docs/oauth.md](docs/oauth.md).

### 3. Development (Live Reload)

Use `mise` to run both the Go server (with Air for hot reloading) and Vite dev server (with Vue HMR) concurrently:

```bash
mise run dev
```

- **Frontend (Vite)**: [http://localhost:5173](http://localhost:5173) (proxies `/api` and `/auth` to Go backend)
- **Backend (Go + Air)**: [http://localhost:8080](http://localhost:8080)

You can also run them individually:
```bash
mise run dev:server    # Air only
mise run dev:frontend  # Vite only
```

### 4. Production Build & Run

To build both artifacts:

```bash
mise run build
# or ./run.sh
```

When running the compiled server directly or with `./run.sh`, the Go server serves the static frontend from `./frontend/dist` on [http://localhost:8080](http://localhost:8080).
---

## Documentation

- [OAuth Configuration & Architecture](docs/oauth.md) — Comprehensive guide covering Google Cloud Console setup, OAuth flow, session handling, security model, and troubleshooting.
- [Project Goals & Specs](docs/goal.md) — Product roadmap and core objectives.
