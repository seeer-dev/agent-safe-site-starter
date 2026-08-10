# Email

Business modules depend on the small `platform/mail.Sender` interface. Production uses Resend; local mode logs.

Persist important business state before sending a notification. If a future feature requires guaranteed delivery/retries, add an outbox for that feature instead of building a global messaging framework prematurely.
