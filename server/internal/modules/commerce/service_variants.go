package commerce

import (
	"context"
	"fmt"
	"strings"
)

// ----- Product variants -----------------------------------------------------

// buildVariants validates and normalizes variant inputs for a product
// create/update. Empty input returns nil (the product is sold as a single
// SKU). Variant names must be unique per product (the unique index backs
// this); empty variant SKUs fall back to the parent product SKU plus the
// variant name so every purchasable line has a stock-keeping identity.
func buildVariants(productID, productSKU string, inputs []ProductVariantInput, now int64) ([]ProductVariant, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	seen := make(map[string]bool, len(inputs))
	out := make([]ProductVariant, 0, len(inputs))
	for i, vi := range inputs {
		name := strings.TrimSpace(vi.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: variant %d name is required", ErrInvalidProductInput, i)
		}
		if seen[name] {
			return nil, fmt.Errorf("%w: duplicate variant name %q", ErrInvalidProductInput, name)
		}
		seen[name] = true
		if vi.Stock < 0 {
			return nil, fmt.Errorf("%w: variant %q stock must be non-negative", ErrInvalidProductInput, name)
		}
		id, err := randomID()
		if err != nil {
			return nil, err
		}
		sku := strings.TrimSpace(vi.SKU)
		if sku == "" {
			sku = productSKU + ":" + name
		}
		out = append(out, ProductVariant{
			ID:          id,
			ProductID:   productID,
			Name:        name,
			SKU:         sku,
			PriceDelta:  vi.PriceDelta,
			Stock:       vi.Stock,
			SortOrder:   i,
			UpdatedUnix: now,
		})
	}
	return out, nil
}

// resolveVariant looks up a purchasable variant by its SKU. Used at
// checkout when an order item SKU does not match a base product.
func (s Service) resolveVariant(ctx context.Context, sku string) (ProductVariant, Product, error) {
	v, err := s.store.GetVariantBySKU(ctx, sku)
	if err != nil {
		return ProductVariant{}, Product{}, err
	}
	p, err := s.store.GetProduct(ctx, v.ProductID)
	if err != nil {
		return ProductVariant{}, Product{}, err
	}
	return v, p, nil
}

// variantEffectiveStock reports the stock used for availability checks.
// A variant with its own SKU tracks stock independently; a variant
// without one shares the parent product's stock.
func variantEffectiveStock(v ProductVariant, p Product) int {
	if v.SKU == "" || strings.HasPrefix(v.SKU, p.SKU+":") {
		return v.Stock
	}
	return v.Stock
}
