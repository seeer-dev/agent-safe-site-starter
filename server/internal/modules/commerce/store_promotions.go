package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Promos ---------------------------------------------------------------

func (s SQLStore) ListPromos(ctx context.Context) ([]Promo, error) {
	query := database.Bind(s.dialect, `SELECT id, code, label, type, value, enabled, starts_unix, expires_unix, updated_unix FROM promos ORDER BY updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPromos(rows)
}

func (s SQLStore) GetActivePromoByCode(ctx context.Context, code string, now int64) (Promo, error) {
	query := database.Bind(s.dialect, `SELECT id, code, label, type, value, enabled, starts_unix, expires_unix, updated_unix
		FROM promos WHERE code = ? AND enabled = ? AND starts_unix <= ? AND (expires_unix = 0 OR expires_unix >= ?) LIMIT 1`)
	var p Promo
	if err := s.db.QueryRowContext(ctx, query, code, true, now, now).Scan(
		&p.ID, &p.Code, &p.Label, &p.Type, &p.Value, &p.Enabled, &p.StartsUnix, &p.ExpiresUnix, &p.UpdatedUnix,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Promo{}, ErrNotFound
		}
		return Promo{}, err
	}
	return p, nil
}

func (s SQLStore) UpsertPromo(ctx context.Context, p Promo) error {
	query := database.Bind(s.dialect, `INSERT INTO promos
		(id, code, label, type, value, enabled, starts_unix, expires_unix, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			code = excluded.code,
			label = excluded.label,
			type = excluded.type,
			value = excluded.value,
			enabled = excluded.enabled,
			starts_unix = excluded.starts_unix,
			expires_unix = excluded.expires_unix,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.Code, p.Label, p.Type, p.Value, p.Enabled, p.StartsUnix, p.ExpiresUnix, p.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert promo: %w", err)
	}
	return nil
}

func (s SQLStore) DeletePromo(ctx context.Context, id string) error {
	query := database.Bind(s.dialect, `DELETE FROM promos WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete promo: %w", err)
	}
	return requireAffected(res)
}
