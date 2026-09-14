package commerce

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	mailplatform "github.com/example/ai-site-starter/server/internal/platform/mail"
)

// ----- Notification templates and delivery ----------------------------------

// NotificationSender is the mail seam commerce uses to deliver order
// lifecycle notifications. It is satisfied by platform/mail.Sender; the
// interface is declared here so the service never depends on a concrete
// provider.
type NotificationSender interface {
	Send(ctx context.Context, message mailplatform.Message) error
}

// WithNotifier returns a Service that delivers order lifecycle
// notifications through the given sender. When nil, every notification
// attempt is logged as "skipped" with provider "none" — the log still
// records the decision so the admin view shows the complete trail.
func (s Service) WithNotifier(sender NotificationSender) Service {
	s.notifier = sender
	return s
}

// notificationCodeForStatus maps an order status transition to the
// notification template code. Returns "" when the transition has no
// customer-facing notification.
func notificationCodeForStatus(status string) string {
	switch status {
	case "shipped":
		return "order_shipped"
	case "delivered", "completed":
		return "order_completed"
	case "cancelled":
		return "order_cancelled"
	default:
		return ""
	}
}

// validTemplateCodes is the closed set of order lifecycle template codes.
var validTemplateCodes = map[string]bool{
	"order_placed":    true,
	"order_paid":      true,
	"order_shipped":   true,
	"order_completed": true,
	"order_cancelled": true,
}

var templatePlaceholder = regexp.MustCompile(`\{\{[a-zA-Z0-9_]+\}\}`)

// renderTemplate substitutes {{key}} placeholders with the given values.
// Unknown placeholders are left intact so staff can preview exactly what
// the template engine recognized.
func renderTemplate(body string, values map[string]string) string {
	return templatePlaceholder.ReplaceAllStringFunc(body, func(m string) string {
		key := m[2 : len(m)-2]
		if v, ok := values[key]; ok {
			return v
		}
		return m
	})
}

// orderTemplateValues builds the placeholder map for an order
// notification.
func orderTemplateValues(o Order) map[string]string {
	return map[string]string{
		"order_id":        o.ID,
		"customer_name":   o.CustomerName,
		"email":           o.Email,
		"status":          o.Status,
		"payment_status":  o.PaymentStatus,
		"total":           strconv.Itoa(o.Total),
		"tracking_number": o.TrackingNumber,
		"shipping_method": o.ShippingMethod,
	}
}

// notifyOrderEvent renders the template for the given code and delivers
// it to the order contact. Every attempt — sent, skipped (no template,
// disabled, no recipient, no sender), or failed — is recorded in
// notification_logs. Errors are logged but never propagated: a mail
// failure must not roll back the order mutation that triggered it.
func (s Service) notifyOrderEvent(ctx context.Context, o Order, code string) {
	if code == "" {
		return
	}
	now := time.Now().Unix()
	log := NotificationLog{
		Code:        code,
		OrderID:     o.ID,
		Recipient:   o.Email,
		Status:      "skipped",
		CreatedUnix: now,
	}
	id, err := randomID()
	if err != nil {
		return
	}
	log.ID = id

	tpl, err := s.store.GetNotificationTemplateByCode(ctx, code)
	if err != nil {
		log.Error = "template not found"
		_ = s.store.InsertNotificationLog(ctx, log)
		return
	}
	if !tpl.IsEnabled {
		log.Error = "template disabled"
		_ = s.store.InsertNotificationLog(ctx, log)
		return
	}
	if o.Email == "" {
		log.Error = "no recipient"
		_ = s.store.InsertNotificationLog(ctx, log)
		return
	}
	values := orderTemplateValues(o)
	log.Subject = renderTemplate(tpl.Subject, values)
	log.Body = renderTemplate(tpl.Body, values)

	if s.notifier == nil {
		log.Error = "no sender configured"
		_ = s.store.InsertNotificationLog(ctx, log)
		return
	}
	err = s.notifier.Send(ctx, mailplatform.Message{
		To:      []string{o.Email},
		Subject: log.Subject,
		Text:    log.Body,
	})
	if err != nil {
		log.Status = "failed"
		log.Provider = "mail"
		log.Error = err.Error()
		_ = s.store.InsertNotificationLog(ctx, log)
		return
	}
	log.Status = "sent"
	log.Provider = "mail"
	_ = s.store.InsertNotificationLog(ctx, log)
}

// NotifyOrderPlaced sends the order_placed notification after order
// creation. Called by the HTTP handler once the order is committed.
func (s Service) NotifyOrderPlaced(ctx context.Context, o Order) {
	s.notifyOrderEvent(ctx, o, "order_placed")
}

// NotifyOrderPaid sends the order_paid notification. Called when payment
// status transitions to paid (ECPay callback or admin mark-paid).
func (s Service) NotifyOrderPaid(ctx context.Context, o Order) {
	s.notifyOrderEvent(ctx, o, "order_paid")
}

// ----- Admin CRUD -------------------------------------------------------------

func (s Service) ListNotificationTemplates(ctx context.Context, principal auth.Principal) ([]NotificationTemplate, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return nil, ErrForbidden
	}
	return s.store.ListNotificationTemplates(ctx)
}

func (s Service) UpsertNotificationTemplate(ctx context.Context, principal auth.Principal, id string, in NotificationTemplateInput) (NotificationTemplate, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return NotificationTemplate{}, ErrForbidden
	}
	code := strings.TrimSpace(in.Code)
	if !validTemplateCodes[code] {
		return NotificationTemplate{}, fmt.Errorf("%w: template code must be one of order_placed, order_paid, order_shipped, order_completed, order_cancelled", ErrInvalidAdminInput)
	}
	if strings.TrimSpace(in.Subject) == "" {
		return NotificationTemplate{}, fmt.Errorf("%w: template subject is required", ErrInvalidAdminInput)
	}
	if id == "" {
		var err error
		id, err = randomID()
		if err != nil {
			return NotificationTemplate{}, err
		}
	}
	t := NotificationTemplate{
		ID:          id,
		Code:        code,
		Name:        strings.TrimSpace(in.Name),
		Subject:     in.Subject,
		Body:        in.Body,
		IsEnabled:   in.IsEnabled,
		UpdatedUnix: time.Now().Unix(),
	}
	if err := s.store.UpsertNotificationTemplate(ctx, t); err != nil {
		return NotificationTemplate{}, err
	}
	return t, nil
}

func (s Service) DeleteNotificationTemplate(ctx context.Context, principal auth.Principal, id string) error {
	if !auth.Can(principal, "twcommerce.admin") {
		return ErrForbidden
	}
	return s.store.DeleteNotificationTemplate(ctx, id)
}

func (s Service) ListNotificationLogs(ctx context.Context, principal auth.Principal, filter NotificationLogFilter) ([]NotificationLog, error) {
	if !auth.Can(principal, "twcommerce.admin") && !auth.Can(principal, "twcommerce.read") {
		return nil, ErrForbidden
	}
	return s.store.ListNotificationLogs(ctx, filter)
}

// AdminStats is the dashboard summary for the admin home view.
type AdminStats struct {
	Products        int `json:"products"`
	Orders          int `json:"orders"`
	PendingComments int `json:"pending_comments"`
	Revenue         int `json:"revenue"`
}

// GetAdminStats aggregates the dashboard counters. twcommerce.read is
// sufficient — the values contain no customer PII.
func (s Service) GetAdminStats(ctx context.Context, principal auth.Principal) (AdminStats, error) {
	if !auth.Can(principal, "twcommerce.read") && !auth.Can(principal, "twcommerce.admin") {
		return AdminStats{}, ErrForbidden
	}
	var stats AdminStats
	var err error
	if stats.Products, err = s.store.CountProducts(ctx); err != nil {
		return stats, err
	}
	if stats.Orders, err = s.store.CountOrders(ctx); err != nil {
		return stats, err
	}
	if stats.PendingComments, err = s.store.CountPendingComments(ctx); err != nil {
		return stats, err
	}
	if stats.Revenue, err = s.store.SumOrderRevenue(ctx); err != nil {
		return stats, err
	}
	return stats, nil
}
