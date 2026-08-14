-- Add request_fingerprint column to orders. This is a SHA-256 hex
-- digest of the canonical normalized OrderInput + memberID, computed
-- at creation time. It is used for idempotency replay validation:
-- when a same-key retry arrives, the server compares the incoming
-- request's fingerprint against the stored one. This allows replay
-- detection to work correctly even when mutable server state (stock,
-- payment method config, fee schedule) has changed since the original
-- creation. The fingerprint captures what the CLIENT requested, not
-- what the server validated against.

ALTER TABLE orders ADD COLUMN request_fingerprint TEXT NOT NULL DEFAULT '';
