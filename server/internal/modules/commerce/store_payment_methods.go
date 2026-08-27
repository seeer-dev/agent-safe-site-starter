package commerce

import (
	"context"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Payment methods ------------------------------------------------------

func (s SQLStore) ListPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	query := database.Bind(s.dialect, `SELECT id, method, provider_label, environment, readiness_status, enabled, updated_unix FROM payment_methods ORDER BY updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PaymentMethod
	for rows.Next() {
		var pm PaymentMethod
		if err := rows.Scan(&pm.ID, &pm.Method, &pm.ProviderLabel, &pm.Environment, &pm.ReadinessStatus, &pm.Enabled, &pm.UpdatedUnix); err != nil {
			return nil, err
		}
		out = append(out, pm)
	}
	return out, rows.Err()
}

func (s SQLStore) UpsertPaymentMethod(ctx context.Context, pm PaymentMethod) error {
	query := database.Bind(s.dialect, `INSERT INTO payment_methods
		(id, method, provider_label, environment, readiness_status, enabled, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			method = excluded.method,
			provider_label = excluded.provider_label,
			environment = excluded.environment,
			readiness_status = excluded.readiness_status,
			enabled = excluded.enabled,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		pm.ID, pm.Method, pm.ProviderLabel, pm.Environment, pm.ReadinessStatus, pm.Enabled, pm.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert payment method: %w", err)
	}
	return nil
}
