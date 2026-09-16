package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v (raw %q)", err, w.Body.String())
	}
	return body
}

// TestJSONSuccessEnvelope pins the success wire shape:
// {"status":"success","data":...} with the endpoint payload preserved
// verbatim inside data and no ok key present.
func TestJSONSuccessEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]any{"order": map[string]any{"id": "TW-1"}})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := decodeBody(t, w)
	if body["status"] != "success" {
		t.Fatalf("status = %v, want success", body["status"])
	}
	if _, present := body["ok"]; present {
		t.Fatalf("ok key present on success: %v", body["ok"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data missing or wrong type: %v", body["data"])
	}
	order, ok := data["order"].(map[string]any)
	if !ok || order["id"] != "TW-1" {
		t.Fatalf("data.order = %v, want preserved payload", data["order"])
	}
	if _, present := body["error"]; present {
		t.Fatalf("error key present on success: %v", body["error"])
	}
}

// TestJSONNon2xxKeepsErrorStatus proves the status discriminator mirrors the
// HTTP status: a value written with a non-2xx status via JSON still emits
// "error" so the flag can never disagree with the transport status.
func TestJSONNon2xxKeepsErrorStatus(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusConflict, map[string]any{"expected_version": 3})
	body := decodeBody(t, w)
	if body["status"] != "error" {
		t.Fatalf("status = %v, want error for non-2xx", body["status"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok || data["expected_version"] != float64(3) {
		t.Fatalf("data = %v, want preserved payload", body["data"])
	}
}

// TestErrorEnvelope pins the failure wire shape:
// {"status":"error","error":{"code":<status code>,"message":<public msg>}}.
func TestErrorEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusForbidden, "forbidden")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	body := decodeBody(t, w)
	if body["status"] != "error" {
		t.Fatalf("status = %v, want error", body["status"])
	}
	if _, present := body["ok"]; present {
		t.Fatalf("ok key present on error: %v", body["ok"])
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error missing or not an object: %v", body["error"])
	}
	if errObj["code"] != "forbidden" {
		t.Fatalf("error.code = %v, want forbidden", errObj["code"])
	}
	if errObj["message"] != "forbidden" {
		t.Fatalf("error.message = %v, want forbidden", errObj["message"])
	}
	if _, present := body["data"]; present {
		t.Fatalf("data key present on error: %v", body["data"])
	}
}

// TestMetaSerialization pins the pagination contract: meta is a top-level
// sibling of data when set and absent otherwise.
func TestMetaSerialization(t *testing.T) {
	w := httptest.NewRecorder()
	writeEnvelope(w, http.StatusOK, Envelope{
		Status: "success",
		Data:   map[string]any{"items": []any{1}},
		Meta:   map[string]any{"page": 1, "per_page": 20, "total": 137, "total_pages": 7},
	})
	body := decodeBody(t, w)
	meta, ok := body["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta missing or wrong type: %v", body["meta"])
	}
	if meta["page"] != float64(1) || meta["per_page"] != float64(20) || meta["total"] != float64(137) || meta["total_pages"] != float64(7) {
		t.Fatalf("meta = %v, want pagination fields", meta)
	}

	w = httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]any{"items": []any{1}})
	body = decodeBody(t, w)
	if _, present := body["meta"]; present {
		t.Fatalf("meta present when unset: %v", body["meta"])
	}
}

// TestErrorCodeMapping pins the status-derived code table used by every
// error site in the codebase.
func TestErrorCodeMapping(t *testing.T) {
	cases := map[int]string{
		http.StatusBadRequest:            "bad_request",
		http.StatusUnauthorized:          "unauthorized",
		http.StatusForbidden:             "forbidden",
		http.StatusNotFound:              "not_found",
		http.StatusMethodNotAllowed:      "method_not_allowed",
		http.StatusConflict:              "conflict",
		http.StatusGone:                  "gone",
		http.StatusRequestEntityTooLarge: "payload_too_large",
		http.StatusUnsupportedMediaType:  "unsupported_media_type",
		http.StatusUnprocessableEntity:   "unprocessable_entity",
		http.StatusTooManyRequests:       "too_many_requests",
		http.StatusInternalServerError:   "internal_error",
		http.StatusBadGateway:            "bad_gateway",
		http.StatusServiceUnavailable:    "service_unavailable",
		http.StatusGatewayTimeout:        "gateway_timeout",
		599:                              "error",
	}
	for status, want := range cases {
		w := httptest.NewRecorder()
		Error(w, status, "m")
		errObj := decodeBody(t, w)["error"].(map[string]any)
		if errObj["code"] != want {
			t.Errorf("status %d code = %v, want %q", status, errObj["code"], want)
		}
	}
}

// TestErrorWithCausePublicBody proves the client sees only the public
// message in the same envelope; the cause never reaches the wire.
func TestErrorWithCausePublicBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	ErrorWithCause(w, r, http.StatusInternalServerError, "temporarily unavailable", errTestCause{})
	body := decodeBody(t, w)
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error missing: %v", body)
	}
	if errObj["code"] != "internal_error" || errObj["message"] != "temporarily unavailable" {
		t.Fatalf("error = %v", errObj)
	}
	if got := w.Body.String(); strings.Contains(got, "secret-cause") {
		t.Fatalf("cause leaked into body: %q", got)
	}
}

type errTestCause struct{}

func (errTestCause) Error() string { return "secret-cause detail" }
