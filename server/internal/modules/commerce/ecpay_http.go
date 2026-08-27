package commerce

import (
	"errors"
	"net/http"
	"strings"

	"github.com/example/ai-site-starter/server/internal/httpx"
)

func (h Handler) PrepareECPayPayment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	token := strings.TrimSpace(r.Header.Get("X-Order-Access-Token"))
	if id == "" || token == "" {
		httpx.Error(w, http.StatusBadRequest, "order id and X-Order-Access-Token are required")
		return
	}
	launch, err := h.service.PrepareECPayPayment(r.Context(), id, token)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.Error(w, http.StatusNotFound, "order not found")
		case errors.Is(err, ErrECPayUnavailable):
			httpx.Error(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, ErrECPayAlreadyPaid), errors.Is(err, ErrECPayCallbackConflict):
			httpx.Error(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrECPayWrongPaymentMethod):
			httpx.Error(w, http.StatusBadRequest, err.Error())
		default:
			httpx.Error(w, http.StatusInternalServerError, "failed to prepare ecpay payment")
		}
		return
	}
	httpx.JSON(w, http.StatusOK, launch)
}

func (h Handler) ReceiveECPayReturn(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid ecpay form")
		return
	}
	ack, err := h.service.ReceiveECPayCallback(r.Context(), r.PostForm)
	if err != nil {
		switch {
		case errors.Is(err, ErrECPayInvalidCallback), errors.Is(err, ErrECPayAmountMismatch):
			httpx.Error(w, http.StatusBadRequest, "invalid ecpay callback")
		case errors.Is(err, ErrECPayCallbackConflict):
			httpx.Error(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrNotFound):
			httpx.Error(w, http.StatusNotFound, "payment attempt not found")
		case errors.Is(err, ErrECPayUnavailable):
			httpx.Error(w, http.StatusServiceUnavailable, err.Error())
		default:
			httpx.Error(w, http.StatusInternalServerError, "failed to process ecpay callback")
		}
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(ack))
}

func (h Handler) ECPayBrowserReturn(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid ecpay form")
		return
	}
	target, err := h.service.ECPayBrowserReturn(r.PostForm)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid ecpay browser return")
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
