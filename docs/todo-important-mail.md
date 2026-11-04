# TODO: Protect important mail (tickets, receipts)

Goal: before an email counts as a cleanup candidate, flag messages worth keeping even when the sender has never been emailed.

Rules:

- **Strictly read-only** (see AGENTS.md). Only search and fetch, using `EXAMINE` / REST list and get calls.
- Flag individual **messages**, never whole senders. A shop sends its receipts and its promotions from the same address.

Check order:

1. Contact check (already done): mail from anyone the user has emailed is excluded.
2. Importance checks below, cheapest first.

## Layer 1: Gmail server-side search (no body downloads)

Gmail already classifies mail. Use its search syntax and store the IDs that match.

- [ ] Run these queries:
  - `category:purchases`
  - `label:^smartlabel_receipt`
  - `label:^smartlabel_travel`
  - `label:^smartlabel_finance`
  - `has:attachment filename:(pdf OR pkpass OR ics)`
  - `receipt OR invoice OR order OR ticket OR booking OR itinerary OR reservation`
- [ ] Before relying on the `^smartlabel_*` queries, paste each into the Gmail search box and confirm it returns what you expect. They are undocumented and could change.
- [ ] Choose a transport:
  - REST: `Users.Messages.List(user).Q(query)`, paged. The returned IDs are hex and match `emails.id`, because IMAP scans store `X-GM-MSGID` in hex (`api/imap_processor.go:278`).
  - IMAP: `UID SEARCH X-GM-RAW "<query>"` in the mailbox already opened with `EXAMINE`. go-imap v1's `SearchCriteria` can't express `X-GM-RAW`, so send it as a raw command.

## Layer 2: local data (fast, already in SQLite)

- [ ] Keep anything that is starred, marked Important, or has a user-created label (check `emails.label_ids`).
- [ ] Run a keyword regex over `subject`:
      `receipt|invoice|order|booking|reservation|itinerary|ticket|e-ticket|confirmation|boarding|tax|statement|warranty|policy`
      (add keywords for any other languages you receive mail in).
  - `snippet` is only filled when the REST fallback is used. The IMAP path leaves it empty, so for IMAP scans only the subject can be matched.
- [ ] Match sender local-part prefixes: `receipts@`, `orders@`, `billing@`, `tickets@`, `invoice@`.
- [ ] Optional: add `BODYSTRUCTURE` to `fetchItems` (`api/imap_processor.go:170`) to detect attachment types (PDF, `.pkpass`, `.ics`) without downloading the attachments.

## Layer 3: only if layers 1 and 2 miss too much

- [ ] Fetch HTML bodies and detect schema.org markup (`Order`, `Invoice`, `FlightReservation`, `EventReservation`, `LodgingReservation`). Very accurate, but every checked body must be downloaded.
- [ ] Use a local language model to classify borderline cases from subject + snippet.

## Integration

- [ ] Add a column `emails.keep_reason TEXT` (e.g. `smartlabel_receipt`, `attachment_pdf`, `subject_keyword`, `starred`). Follow the existing `ALTER TABLE ... ADD COLUMN` backwards-compatibility pattern in `api/storage.go`.
- [ ] Exclude rows where `keep_reason IS NOT NULL` from the candidate queries:
  - candidate count in `GetStats` (`api/storage.go` ~757)
  - `GetCandidateSenders`
  - the `candidates*` filters in `GetEmails`
- [ ] Add a "Possibly important" list or filter in the UI, so flagged mail is visible instead of silently hidden.
- [ ] Tests: a receipt from a never-emailed sender is excluded from candidates, while a promotion from the same sender is still a candidate.
