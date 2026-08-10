package contact

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/httpx"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	mailplatform "github.com/example/ai-site-starter/server/internal/platform/mail"
)

type Input struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
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

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler { return Handler{service: service} }

func (h Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var input Input
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Submit(r.Context(), input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]bool{"ok": true})
}
