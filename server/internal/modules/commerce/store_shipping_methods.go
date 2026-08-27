package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Shipping methods -----------------------------------------------------

func (s SQLStore) ListShippingMethods(ctx context.Context) ([]ShippingMethod, error) {
	query := database.Bind(s.dialect, `SELECT id, method, label, description, fee, free_threshold, enabled, sort_order, version, updated_unix
		FROM shipping_methods ORDER BY sort_order ASC, method ASC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ShippingMethod, 0)
	for rows.Next() {
		m, err := scanShippingMethod(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s SQLStore) GetShippingMethod(ctx context.Context, id string) (ShippingMethod, error) {
	query := database.Bind(s.dialect, `SELECT id, method, label, description, fee, free_threshold, enabled, sort_order, version, updated_unix
		FROM shipping_methods WHERE id = ?`)
	row := s.db.QueryRowContext(ctx, query, id)
	m, err := scanShippingMethod(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ShippingMethod{}, ErrNotFound
		}
		return ShippingMethod{}, err
	}
	return m, nil
}

func (s SQLStore) CreateShippingMethod(ctx context.Context, m ShippingMethod) error {
	var threshold any
	if m.FreeThreshold != nil {
		threshold = *m.FreeThreshold
	}
	query := database.Bind(s.dialect, `INSERT INTO shipping_methods
		(id, method, label, description, fee, free_threshold, enabled, sort_order, version, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.db.ExecContext(ctx, query,
		m.ID, m.Method, m.Label, m.Description, m.Fee, threshold, m.Enabled, m.SortOrder, m.Version, m.UpdatedUnix)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return ErrDuplicateShippingMethod
		}
		return fmt.Errorf("create shipping method: %w", err)
	}
	return nil
}

func (s SQLStore) UpdateShippingMethod(ctx context.Context, m ShippingMethod, expectedVersion int) error {
	var threshold any
	if m.FreeThreshold != nil {
		threshold = *m.FreeThreshold
	}
	// method is immutable: it is not included in the SET list.
	query := database.Bind(s.dialect, `UPDATE shipping_methods SET
		label = ?, description = ?, fee = ?, free_threshold = ?, enabled = ?,
		sort_order = ?, version = version + 1, updated_unix = ?
		WHERE id = ? AND version = ?`)
	res, err := s.db.ExecContext(ctx, query,
		m.Label, m.Description, m.Fee, threshold, m.Enabled, m.SortOrder, m.UpdatedUnix, m.ID, expectedVersion)
	if err != nil {
		return fmt.Errorf("update shipping method: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		_, getErr := s.GetShippingMethod(ctx, m.ID)
		if errors.Is(getErr, ErrNotFound) {
			return ErrNotFound
		}
		if getErr != nil {
			return getErr
		}
		return ErrStaleShippingMethod
	}
	return nil
}

type shippingMethodScanner interface {
	Scan(dest ...any) error
}

func scanShippingMethod(row shippingMethodScanner) (ShippingMethod, error) {
	var m ShippingMethod
	var threshold sql.NullInt64
	var enabled bool
	if err := row.Scan(
		&m.ID, &m.Method, &m.Label, &m.Description, &m.Fee, &threshold,
		&enabled, &m.SortOrder, &m.Version, &m.UpdatedUnix,
	); err != nil {
		return ShippingMethod{}, err
	}
	m.Enabled = enabled
	if threshold.Valid {
		v := int(threshold.Int64)
		m.FreeThreshold = &v
	}
	return m, nil
}
