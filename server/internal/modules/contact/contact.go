package contact

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/httpx"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	mailplatform "github.com/example/ai-site-starter/server/internal/platform/mail"
	"github.com/example/ai-site-starter/server/internal/platform/turnstile"
)

type Input struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
	// TurnstileToken is the Cloudflare Turnstile response token for the
	// "contact" widget action. Verified before the inquiry is persisted.
	TurnstileToken string `json:"turnstile_token"`
}

type Store struct {
	db      *sql.DB
	dialect database.Dialect
}

func NewStore(db *sql.DB, dialect database.Dialect) Store {
	return Store{db: db, dialect: dialect}
}

func (s Store) Insert(ctx context.Context, in Input) error {
	query := database.Bind(s.dialect, `INSERT INTO contact_inquiries (name, email, message, created_unix) VALUES (?, ?, ?, ?)`)
	_, err := s.db.ExecContext(ctx, query, in.Name, in.Email, in.Message, time.Now().Unix())
	return err
}

type Service struct {
	store    Store
	mailer   mailplatform.Sender
	notifyTo string
}

func NewService(store Store, mailer mailplatform.Sender, notifyTo string) Service {
	return Service{store: store, mailer: mailer, notifyTo: notifyTo}
}

func (s Service) Submit(ctx context.Context, in Input) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)
	in.Message = strings.TrimSpace(in.Message)
	if in.Name == "" || in.Email == "" || in.Message == "" {
		return fmt.Errorf("name, email, and message are required")
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return fmt.Errorf("invalid email")
	}
	if len(in.Name) > 120 || len(in.Email) > 254 || len(in.Message) > 5000 {
		return fmt.Errorf("contact fields exceed starter limits")
	}
	if err := s.store.Insert(ctx, in); err != nil {
		return fmt.Errorf("save inquiry: %w", err)
	}
	if s.notifyTo == "" {
		return nil
	}
	return s.mailer.Send(ctx, mailplatform.Message{
		To:      []string{s.notifyTo},
		Subject: "New website inquiry from " + in.Name,
		Text:    fmt.Sprintf("Name: %s\nEmail: %s\n\n%s", in.Name, in.Email, in.Message),
		HTML: fmt.Sprintf("<p><strong>Name:</strong> %s</p><p><strong>Email:</strong> %s</p><p>%s</p>",
			html.EscapeString(in.Name), html.EscapeString(in.Email), strings.ReplaceAll(html.EscapeString(in.Message), "\n", "<br>")),
	})
}

// TurnstileVerifier is the smallest interface the contact handler needs
// from the turnstile platform package — bootstrap owns the concrete wiring.
type TurnstileVerifier interface {
	Verify(ctx context.Context, token, action string) error
}

type Handler struct {
	service   Service
	turnstile TurnstileVerifier
}

func NewHandler(service Service) Handler { return Handler{service: service} }

// WithTurnstile wires the side-effect verifier. Bootstrap always supplies
// one; a nil verifier means the check is skipped (unit tests only).
func (h Handler) WithTurnstile(v TurnstileVerifier) Handler {
	h.turnstile = v
	return h
}

func (h Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var input Input
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.turnstile != nil {
		if err := h.turnstile.Verify(r.Context(), input.TurnstileToken, "contact"); err != nil {
			if errors.Is(err, turnstile.ErrUnavailable) {
				httpx.Error(w, http.StatusServiceUnavailable, "service unavailable")
				return
			}
			httpx.Error(w, http.StatusForbidden, "verification failed")
			return
		}
	}
	if err := h.service.Submit(r.Context(), input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]bool{"ok": true})
}
