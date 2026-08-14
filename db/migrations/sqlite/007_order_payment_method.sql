-- Add payment_method column to orders so the server can persist the
-- customer's selected payment method for fulfillment. The server
-- validates the payment method against the payment_methods table
-- (enabled + readiness_status="ready") before accepting an order.
-- The browser is not the authority for payment method availability.

ALTER TABLE orders ADD COLUMN payment_method TEXT NOT NULL DEFAULT '';
