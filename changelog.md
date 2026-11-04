# Changelog

## Unreleased

- Anyone you have sent an email to is now treated as a real contact and is never shown. Sender lists, email lists, single-email lookups, and deletion candidates only include received mail from people you have never emailed.
- The dashboard now shows what share of all scanned emails and all mailbox space comes from people you have never emailed. Totals still count every scanned email.
- Removed the "All Senders" / "Reciprocal Contacts" sender filters, the "All Emails" / "One-Way Senders" email filters, and the "Most Sent" sort option.
- API: `/api/inbox/contacts` no longer accepts `filter`. Contact entries no longer include `sentCount`, `firstSent`, `lastSent`, or `neverEmailed`. Emails no longer include `senderNeverEmailed`. `/api/emails` accepts only the `candidates[_3m|_6m|_1y]` filters. `/api/emails/{id}` returns 404 for sent mail and for mail from your contacts.
- API: in `/api/inbox/stats`, `neverEmailedCount` is renamed to `neverEmailedSenders`, and new `neverEmailedEmails` and `neverEmailedSizeBytes` fields were added.
- Added the `emails(user_id, is_sent, from_address)` index so the never-emailed join stays fast on large mailboxes.
