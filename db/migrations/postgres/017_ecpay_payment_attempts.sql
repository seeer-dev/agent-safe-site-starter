CREATE TABLE IF NOT EXISTS ecpay_payment_attempts (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
  merchant_trade_no TEXT NOT NULL UNIQUE,
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL DEFAULT 'TWD' CHECK (currency = 'TWD'),
  status TEXT NOT NULL DEFAULT 'prepared',
  provider_trade_no TEXT NOT NULL DEFAULT '',
  rtn_code TEXT NOT NULL DEFAULT '',
  callback_fingerprint TEXT NOT NULL DEFAULT '',
  created_unix INTEGER NOT NULL,
  updated_unix INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ecpay_payment_attempts_status ON ecpay_payment_attempts (status);
