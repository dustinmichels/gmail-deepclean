# Gmail DeepClean

A local-first tool for analyzing and cleaning your Gmail inbox using the official Gmail API with strictly **read-only** permissions.

## Features

- **Strictly Read-Only**: Requests only `https://www.googleapis.com/auth/gmail.readonly`—never modifies, deletes, or sends emails.
- **Inbox Analytics**: Identifies high-volume senders, contacts who send emails you never reply to, and detailed interaction metrics.
- **Local & Private**: All data and OAuth tokens are stored in a local SQLite database (`data/deepclean.db`).
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

> For step-by-step instructions on setting up Google Cloud Console credentials, see [docs/oauth.md](docs/oauth.md).

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
