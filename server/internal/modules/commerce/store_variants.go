package commerce

import (
	"context"
	"fmt"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Product variants -----------------------------------------------------

func (s SQLStore) ListProductVariants(ctx context.Context, productID string) ([]ProductVariant, error) {
	query := database.Bind(s.dialect, `SELECT id, product_id, name, sku, price_delta, stock, sort_order, updated_unix
		FROM product_variants WHERE product_id = ? ORDER BY sort_order, id`)
	rows, err := s.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("list product variants: %w", err)
	}
	defer rows.Close()
	var out []ProductVariant
	for rows.Next() {
		var v ProductVariant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.Name, &v.SKU, &v.PriceDelta, &v.Stock, &v.SortOrder, &v.UpdatedUnix); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ReplaceProductVariants deletes all variant rows for the product and
// inserts the given set in a single transaction.
func (s SQLStore) ReplaceProductVariants(ctx context.Context, productID string, variants []ProductVariant) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace variants tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	deleteQuery := database.Bind(s.dialect, `DELETE FROM product_variants WHERE product_id = ?`)
	if _, err := tx.ExecContext(ctx, deleteQuery, productID); err != nil {
		return fmt.Errorf("delete old variants: %w", err)
	}

	insertQuery := database.Bind(s.dialect, `INSERT INTO product_variants (id, product_id, name, sku, price_delta, stock, sort_order, updated_unix) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	for i, v := range variants {
		if _, err := tx.ExecContext(ctx, insertQuery, v.ID, productID, v.Name, v.SKU, v.PriceDelta, v.Stock, i, v.UpdatedUnix); err != nil {
			return fmt.Errorf("insert variant %d: %w", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace variants: %w", err)
	}
	return nil
}
