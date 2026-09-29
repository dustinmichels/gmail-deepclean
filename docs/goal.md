# Goal

The app is to help users deepclean their gmail, using the API.

I will run it locally on my laptop, and use the browser for UI.

User must authenticate with gmail, then the app can use gmail APIs to begin looking at sent and recieved emails.

The first goal is create a list of all email addresses the user has sent and received email from, and a count.

Especially: I want to identify people that send me email who I have never emailed.

## TODO

1.  Frontend / Backend Disconnect:

- The UI does not expose triggers for inbox processing, sender breakdown charts/tables, or batch actions by sender.
- The basic list view only displays raw Gmail message IDs rather than sender, subject, date, or preview snippet.

2.  API Quota & Concurrency Risks:

- InboxProcessor.processInbox() spins up 100 unbounded concurrent goroutines per page calling Messages.Get. For
  users with large inboxes, this will rapidly exhaust Google's rate limits (per-user quota limits / HTTP 429)
  without a worker pool or rate limiter.

3.  Ephemeral State:

- Processor status and email stats live in memory in Go variables. Server restarts wipe all progress.

4.  Security / Production Readiness:

- Hardcoded OAuth CSRF state (oauthStateString = "random-state-string").
- OAuth tokens are passed as raw JSON strings in the Authorization header and stored unencrypted in browser
  localStorage.
- User identity is tied to the first 10 characters of the access token (token.AccessToken[:10]) rather than the
  authenticated Google user ID or email.

5.  No Automated Tests or Env Template:

- No Go _test.go files or frontend test suites.
- Requires .env with GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and REDIRECT_URL, but no .env.example is provided in
  the repository.
