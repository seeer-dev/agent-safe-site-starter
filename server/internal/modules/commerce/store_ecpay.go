package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

func scanECPayAttempt(row interface{ Scan(...any) error }) (ECPayPaymentAttempt, error) {
	var a ECPayPaymentAttempt
	err := row.Scan(&a.ID, &a.OrderID, &a.MerchantTradeNo, &a.Amount, &a.Currency, &a.Status, &a.ProviderTradeNo, &a.RtnCode, &a.CallbackFingerprint, &a.CreatedUnix, &a.UpdatedUnix)
	if errors.Is(err, sql.ErrNoRows) {
		return ECPayPaymentAttempt{}, ErrNotFound
	}
	return a, err
}

func (s SQLStore) EnsureECPayAttempt(ctx context.Context, attempt ECPayPaymentAttempt) (ECPayPaymentAttempt, error) {
	query := database.Bind(s.dialect, `INSERT INTO ecpay_payment_attempts (id, order_id, merchant_trade_no, amount, currency, status, provider_trade_no, rtn_code, callback_fingerprint, created_unix, updated_unix) VALUES (?, ?, ?, ?, ?, ?, '', '', '', ?, ?)`)
	_, err := s.db.ExecContext(ctx, query, attempt.ID, attempt.OrderID, attempt.MerchantTradeNo, attempt.Amount, attempt.Currency, attempt.Status, attempt.CreatedUnix, attempt.UpdatedUnix)
	if err == nil {
		return attempt, nil
	}
	if !database.IsUniqueViolation(err) {
		return ECPayPaymentAttempt{}, fmt.Errorf("insert ecpay attempt: %w", err)
	}
	query = database.Bind(s.dialect, `SELECT id, order_id, merchant_trade_no, amount, currency, status, provider_trade_no, rtn_code, callback_fingerprint, created_unix, updated_unix FROM ecpay_payment_attempts WHERE order_id = ? LIMIT 1`)
	return scanECPayAttempt(s.db.QueryRowContext(ctx, query, attempt.OrderID))
}

func (s SQLStore) GetECPayAttemptByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (ECPayPaymentAttempt, error) {
	query := database.Bind(s.dialect, `SELECT id, order_id, merchant_trade_no, amount, currency, status, provider_trade_no, rtn_code, callback_fingerprint, created_unix, updated_unix FROM ecpay_payment_attempts WHERE merchant_trade_no = ? LIMIT 1`)
	return scanECPayAttempt(s.db.QueryRowContext(ctx, query, merchantTradeNo))
}

func (s SQLStore) ClaimECPayCallback(ctx context.Context, merchantTradeNo, callbackFingerprint, providerTradeNo, rtnCode, status string, captured bool, updatedUnix int64) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	selectQuery := database.Bind(s.dialect, `SELECT id, order_id, merchant_trade_no, amount, currency, status, provider_trade_no, rtn_code, callback_fingerprint, created_unix, updated_unix FROM ecpay_payment_attempts WHERE merchant_trade_no = ? LIMIT 1`)
	current, err := scanECPayAttempt(tx.QueryRowContext(ctx, selectQuery, merchantTradeNo))
	if err != nil {
		return false, err
	}
	if current.CallbackFingerprint != "" {
		if current.CallbackFingerprint == callbackFingerprint && current.ProviderTradeNo == providerTradeNo && current.RtnCode == rtnCode && current.Status == status {
			if err := tx.Commit(); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, ErrECPayCallbackConflict
	}

	updateAttempt := database.Bind(s.dialect, `UPDATE ecpay_payment_attempts SET status = ?, provider_trade_no = ?, rtn_code = ?, callback_fingerprint = ?, updated_unix = ? WHERE merchant_trade_no = ? AND callback_fingerprint = ''`)
	res, err := tx.ExecContext(ctx, updateAttempt, status, providerTradeNo, rtnCode, callbackFingerprint, updatedUnix, merchantTradeNo)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		current, err = scanECPayAttempt(tx.QueryRowContext(ctx, selectQuery, merchantTradeNo))
		if err != nil {
			return false, err
		}
		if current.CallbackFingerprint == callbackFingerprint && current.ProviderTradeNo == providerTradeNo && current.RtnCode == rtnCode && current.Status == status {
			if err := tx.Commit(); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, ErrECPayCallbackConflict
	}

	if captured {
		updateOrder := database.Bind(s.dialect, `UPDATE orders SET payment_status = 'paid', payment_intent_id = ?, version = version + 1, updated_unix = ? WHERE id = ? AND payment_status <> 'paid'`)
		orderRes, err := tx.ExecContext(ctx, updateOrder, merchantTradeNo, updatedUnix, current.OrderID)
		if err != nil {
			return false, err
		}
		orderAffected, err := orderRes.RowsAffected()
		if err != nil {
			return false, err
		}
		if orderAffected == 1 {
			var seq int
			seqQuery := database.Bind(s.dialect, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM order_events WHERE order_id = ?`)
			if err := tx.QueryRowContext(ctx, seqQuery, current.OrderID).Scan(&seq); err != nil {
				return false, err
			}
			eventQuery := database.Bind(s.dialect, `INSERT INTO order_events (id, order_id, event_type, sequence, actor_user_id, from_status, to_status, reason, created_unix) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
			eventID := current.OrderID + ":ecpay:" + callbackFingerprint[:12]
			if _, err := tx.ExecContext(ctx, eventQuery, eventID, current.OrderID, "payment_status", seq, "ecpay", "unpaid", "paid", "verified ecpay callback", updatedUnix); err != nil {
				return false, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return false, nil
}
