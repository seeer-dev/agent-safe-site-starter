package commerce

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProductMarshalJSONExposesImagesArray(t *testing.T) {
	t.Parallel()
	p := Product{ID: "p1", Images: `["/one.webp","/two.webp"]`}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body struct {
		Images []string `json:"images"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Images) != 2 || body.Images[0] != "/one.webp" || body.Images[1] != "/two.webp" {
		t.Fatalf("images = %#v, want ordered image array", body.Images)
	}
}

// TestProductMarshalJSONDoesNotLeakObjectKeys proves that the public
// Product JSON shape never includes product_images or object_key
// fields, even when ProductImages is populated. Verified object keys
// are internal-only; public responses expose only derived URL fields.
func TestProductMarshalJSONDoesNotLeakObjectKeys(t *testing.T) {
	t.Parallel()
	p := Product{
		ID: "p1", Images: `["https://cdn.example/img.jpg"]`,
		ProductImages: []ProductImage{
			{ID: "img1", ProductID: "p1", ObjectKey: "verified/product-images/owner/abc123.jpg"},
		},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, exists := body["product_images"]; exists {
		t.Fatal("public Product JSON must NOT contain product_images field")
	}
	if _, exists := body["object_key"]; exists {
		t.Fatal("public Product JSON must NOT contain object_key field")
	}
}

// TestProductInputRejectsImageFields proves that the legacy image/images
// fields are absent from ProductInput. Since httpx.DecodeJSON uses
// DisallowUnknownFields, any payload containing these fields is rejected
// at the handler boundary. This test uses json.Decoder directly to
// prove the rejection.
func TestProductInputRejectsImageFields(t *testing.T) {
	t.Parallel()
	dec := json.NewDecoder(strings.NewReader(`{"image":"https://attacker.example/x.jpg","sku":"S1","name":"N","slug":"s"}`))
	dec.DisallowUnknownFields()
	var in ProductInput
	if err := dec.Decode(&in); err == nil {
		t.Fatal("payload with image field accepted; want rejection via DisallowUnknownFields")
	}

	dec2 := json.NewDecoder(strings.NewReader(`{"images":["https://attacker.example/x.jpg"],"sku":"S1","name":"N","slug":"s"}`))
	dec2.DisallowUnknownFields()
	var in2 ProductInput
	if err := dec2.Decode(&in2); err == nil {
		t.Fatal("payload with images field accepted; want rejection via DisallowUnknownFields")
	}
}

func TestEncodeProductImagesUsesEmptyArrayForNil(t *testing.T) {
	t.Parallel()
	if got := encodeProductImages(nil); got != "[]" {
		t.Fatalf("encodeProductImages(nil) = %q, want []", got)
	}
}
