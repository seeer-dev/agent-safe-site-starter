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

// TestJSONSuccessEnvelope pins the success wire shape: {"ok":true,"data":...}
// with the endpoint payload preserved verbatim inside data.
func TestJSONSuccessEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]any{"order": map[string]any{"id": "TW-1"}})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := decodeBody(t, w)
	if body["ok"] != true {
		t.Fatalf("ok = %v, want true", body["ok"])
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

// TestJSONNon2xxKeepsOkFalse proves the ok discriminator mirrors the status:
// a value written with a non-2xx status via JSON still emits ok:false so the
// flag can never disagree with the transport status.
func TestJSONNon2xxKeepsOkFalse(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusConflict, map[string]any{"expected_version": 3})
	body := decodeBody(t, w)
	if body["ok"] != false {
		t.Fatalf("ok = %v, want false for non-2xx", body["ok"])
	}
	data, ok := body["data"].(map[string]any)
	if !ok || data["expected_version"] != float64(3) {
		t.Fatalf("data = %v, want preserved payload", body["data"])
	}
}

// TestErrorEnvelope pins the failure wire shape:
// {"ok":false,"error":{"code":<status code>,"message":<public msg>}}.
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
	if body["ok"] != false {
		t.Fatalf("ok = %v, want false", body["ok"])
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
