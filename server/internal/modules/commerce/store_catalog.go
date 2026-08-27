package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// ----- Products -------------------------------------------------------------

func (s SQLStore) ListProducts(ctx context.Context, filter ProductFilter) ([]Product, error) {
	var (
		clauses []string
		args    []any
	)
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Category != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, filter.Category)
	}
	query := `SELECT id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix FROM products`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY updated_unix DESC"
	query = database.Bind(s.dialect, query)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

func (s SQLStore) ListPublishedProducts(ctx context.Context) ([]Product, error) {
	query := database.Bind(s.dialect, `SELECT id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix
		FROM products WHERE status IN ('active','low_stock','out_of_stock') ORDER BY updated_unix DESC`)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

func (s SQLStore) GetProduct(ctx context.Context, id string) (Product, error) {
	query := database.Bind(s.dialect, `SELECT id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix
		FROM products WHERE id = ? LIMIT 1`)
	return scanProductRow(s.db.QueryRowContext(ctx, query, id))
}

func (s SQLStore) GetProductBySlug(ctx context.Context, slug string) (Product, error) {
	query := database.Bind(s.dialect, `SELECT id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix
		FROM products WHERE slug = ? AND status IN ('active','low_stock','out_of_stock') LIMIT 1`)
	return scanProductRow(s.db.QueryRowContext(ctx, query, slug))
}

func (s SQLStore) UpsertProduct(ctx context.Context, p Product) error {
	query := database.Bind(s.dialect, `INSERT INTO products
		(id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			sku = excluded.sku,
			name = excluded.name,
			slug = excluded.slug,
			description = excluded.description,
			long_description = excluded.long_description,
			image = excluded.image,
			images = excluded.images,
			category = excluded.category,
			status = excluded.status,
			material = excluded.material,
			origin = excluded.origin,
			price = excluded.price,
			original_price = excluded.original_price,
			stock = excluded.stock,
			tag = excluded.tag,
			rating = excluded.rating,
			reviews_count = excluded.reviews_count,
			updated_unix = excluded.updated_unix`)
	_, err := s.db.ExecContext(ctx, query,
		p.ID, p.SKU, p.Name, p.Slug, p.Description, p.LongDescription, p.Image, p.Images,
		p.Category, p.Status, p.Material, p.Origin, p.Price, p.OriginalPrice, p.Stock,
		p.Tag, p.Rating, p.ReviewsCount, p.UpdatedUnix)
	if err != nil {
		return fmt.Errorf("upsert product: %w", err)
	}
	return nil
}

func (s SQLStore) UpdateProductStatus(ctx context.Context, id, status string, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE products SET status = ?, updated_unix = ? WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, query, status, updatedUnix, id)
	if err != nil {
		return fmt.Errorf("update product status: %w", err)
	}
	return requireAffected(res)
}

func (s SQLStore) DeleteProduct(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete product: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	oldKeys, err := productImageKeysTx(ctx, tx, s.dialect, id)
	if err != nil {
		return err
	}
	query := database.Bind(s.dialect, `DELETE FROM products WHERE id = ?`)
	res, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if err := requireAffected(res); err != nil {
		return err
	}
	if err := markMediaAssociationsTx(ctx, tx, s.dialect, oldKeys, nil, time.Now().Unix()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete product: %w", err)
	}
	return nil
}

func (s SQLStore) BulkUpdateProductStatus(ctx context.Context, ids []string, status string, updatedUnix int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+2)
	args = append(args, status, updatedUnix)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := database.Bind(s.dialect, `UPDATE products SET status = ?, updated_unix = ? WHERE id IN (`+strings.Join(placeholders, ", ")+`)`)
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("bulk update product status: %w", err)
	}
	return nil
}

func (s SQLStore) DecrementProductStock(ctx context.Context, sku string, qty int, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE products SET stock = stock - ?, updated_unix = ? WHERE sku = ? AND stock >= ?`)
	res, err := s.db.ExecContext(ctx, query, qty, updatedUnix, sku, qty)
	if err != nil {
		return fmt.Errorf("decrement stock: %w", err)
	}
	return requireAffected(res)
}

func (s SQLStore) IncrementProductStock(ctx context.Context, sku string, qty int, updatedUnix int64) error {
	query := database.Bind(s.dialect, `UPDATE products SET stock = stock + ?, updated_unix = ? WHERE sku = ?`)
	res, err := s.db.ExecContext(ctx, query, qty, updatedUnix, sku)
	if err != nil {
		return fmt.Errorf("increment stock: %w", err)
	}
	return requireAffected(res)
}

// ----- Product images -------------------------------------------------------

func (s SQLStore) ListProductImages(ctx context.Context, productID string) ([]ProductImage, error) {
	query := database.Bind(s.dialect, `SELECT id, product_id, object_key, alt_text, sort_order, created_unix
		FROM product_images WHERE product_id = ? ORDER BY sort_order, created_unix`)
	rows, err := s.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("list product images: %w", err)
	}
	defer rows.Close()
	var out []ProductImage
	for rows.Next() {
		var img ProductImage
		if err := rows.Scan(&img.ID, &img.ProductID, &img.ObjectKey, &img.AltText, &img.SortOrder, &img.CreatedUnix); err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

// ReplaceProductImages deletes all existing product_images rows for the
// given product_id and inserts the new set in a single transaction.
// Sort order is derived from the slice order. This is atomic: if any
// insert fails, the delete is rolled back.
func (s SQLStore) ReplaceProductImages(ctx context.Context, productID string, images []ProductImage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace product images tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	oldKeys, err := productImageKeysTx(ctx, tx, s.dialect, productID)
	if err != nil {
		return err
	}

	deleteQuery := database.Bind(s.dialect, `DELETE FROM product_images WHERE product_id = ?`)
	if _, err := tx.ExecContext(ctx, deleteQuery, productID); err != nil {
		return fmt.Errorf("delete old product images: %w", err)
	}

	insertQuery := database.Bind(s.dialect, `INSERT INTO product_images (id, product_id, object_key, alt_text, sort_order, created_unix) VALUES (?, ?, ?, ?, ?, ?)`)
	for i, img := range images {
		if _, err := tx.ExecContext(ctx, insertQuery, img.ID, productID, img.ObjectKey, img.AltText, i, img.CreatedUnix); err != nil {
			if database.IsForeignKeyViolation(err) {
				return ErrUnverifiedMedia
			}
			return fmt.Errorf("insert product image %d: %w", i, err)
		}
	}
	if err := markMediaAssociationsTx(ctx, tx, s.dialect, oldKeys, images, time.Now().Unix()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace product images: %w", err)
	}
	return nil
}

// UpsertProductWithImages atomically upserts the product and replaces
// all product_images in a single transaction. If the image replacement
// fails, the product upsert is rolled back. This prevents partial
// success where a product is updated but its images are not.
func (s SQLStore) UpsertProductWithImages(ctx context.Context, p Product, images []ProductImage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert product with images tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	upsertQuery := database.Bind(s.dialect, `INSERT INTO products
		(id, sku, name, slug, description, long_description, image, images, category, status, material, origin, price, original_price, stock, tag, rating, reviews_count, updated_unix)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			sku = excluded.sku,
			name = excluded.name,
			slug = excluded.slug,
			description = excluded.description,
			long_description = excluded.long_description,
			image = excluded.image,
			images = excluded.images,
			category = excluded.category,
			status = excluded.status,
			material = excluded.material,
			origin = excluded.origin,
			price = excluded.price,
			original_price = excluded.original_price,
			stock = excluded.stock,
			tag = excluded.tag,
			rating = excluded.rating,
			reviews_count = excluded.reviews_count,
			updated_unix = excluded.updated_unix`)
	if _, err := tx.ExecContext(ctx, upsertQuery,
		p.ID, p.SKU, p.Name, p.Slug, p.Description, p.LongDescription, p.Image, p.Images,
		p.Category, p.Status, p.Material, p.Origin, p.Price, p.OriginalPrice, p.Stock,
		p.Tag, p.Rating, p.ReviewsCount, p.UpdatedUnix); err != nil {
		return fmt.Errorf("upsert product in tx: %w", err)
	}
	oldKeys, err := productImageKeysTx(ctx, tx, s.dialect, p.ID)
	if err != nil {
		return err
	}

	deleteQuery := database.Bind(s.dialect, `DELETE FROM product_images WHERE product_id = ?`)
	if _, err := tx.ExecContext(ctx, deleteQuery, p.ID); err != nil {
		return fmt.Errorf("delete old product images in tx: %w", err)
	}

	insertQuery := database.Bind(s.dialect, `INSERT INTO product_images (id, product_id, object_key, alt_text, sort_order, created_unix) VALUES (?, ?, ?, ?, ?, ?)`)
	for i, img := range images {
		if _, err := tx.ExecContext(ctx, insertQuery, img.ID, p.ID, img.ObjectKey, img.AltText, i, img.CreatedUnix); err != nil {
			if database.IsForeignKeyViolation(err) {
				return ErrUnverifiedMedia
			}
			return fmt.Errorf("insert product image %d in tx: %w", i, err)
		}
	}
	associationUnix := time.Now().Unix()
	if err := markMediaAssociationsTx(ctx, tx, s.dialect, oldKeys, images, associationUnix); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert product with images: %w", err)
	}
	return nil
}

func productImageKeysTx(ctx context.Context, tx *sql.Tx, dialect database.Dialect, productID string) ([]string, error) {
	query := database.Bind(dialect, `SELECT object_key FROM product_images WHERE product_id = ?`)
	rows, err := tx.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("list old product image keys: %w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func markMediaAssociationsTx(ctx context.Context, tx *sql.Tx, dialect database.Dialect, oldKeys []string, images []ProductImage, nowUnix int64) error {
	markLinked := database.Bind(dialect, `UPDATE media_assets SET unassociated_since_unix = 0 WHERE object_key = ? AND state = 'active'`)
	for _, img := range images {
		if _, err := tx.ExecContext(ctx, markLinked, img.ObjectKey); err != nil {
			return fmt.Errorf("protect associated media: %w", err)
		}
	}
	markUnlinked := database.Bind(dialect, `UPDATE media_assets SET unassociated_since_unix = ?
		WHERE object_key = ? AND state = 'active'
		AND NOT EXISTS (SELECT 1 FROM product_images pi WHERE pi.object_key = media_assets.object_key)`)
	for _, key := range oldKeys {
		if _, err := tx.ExecContext(ctx, markUnlinked, nowUnix, key); err != nil {
			return fmt.Errorf("start media gc grace period: %w", err)
		}
	}
	return nil
}
