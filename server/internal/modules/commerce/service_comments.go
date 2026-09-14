package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

// ----- Product comments -----------------------------------------------------

// ErrInvalidCommentInput is returned when a public comment submission is
// blank, over-long, or carries an out-of-range rating.
var ErrInvalidCommentInput = errors.New("invalid comment input")

const (
	maxCommentNicknameLen = 40
	maxCommentContentLen  = 2000
)

var validCommentStatuses = map[string]bool{
	"pending":  true,
	"approved": true,
	"rejected": true,
}

// ListApprovedComments returns the public view of a product's approved
// comments, newest first.
func (s Service) ListApprovedComments(ctx context.Context, productID string) ([]ProductComment, error) {
	return s.store.ListProductComments(ctx, productID, "approved")
}

// SubmitComment is the public submission path. Comments are always
// persisted as pending — only staff moderation promotes them to approved.
// The product must exist and be publicly visible.
func (s Service) SubmitComment(ctx context.Context, productID string, in CommentInput) (ProductComment, error) {
	if _, err := s.store.GetProduct(ctx, productID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ProductComment{}, ErrNotFound
		}
		return ProductComment{}, err
	}
	nickname := strings.TrimSpace(in.Nickname)
	content := strings.TrimSpace(in.Content)
	if nickname == "" || content == "" {
		return ProductComment{}, fmt.Errorf("%w: nickname and content are required", ErrInvalidCommentInput)
	}
	if len(nickname) > maxCommentNicknameLen {
		return ProductComment{}, fmt.Errorf("%w: nickname exceeds %d characters", ErrInvalidCommentInput, maxCommentNicknameLen)
	}
	if len(content) > maxCommentContentLen {
		return ProductComment{}, fmt.Errorf("%w: content exceeds %d characters", ErrInvalidCommentInput, maxCommentContentLen)
	}
	if in.Rating != nil && (*in.Rating < 1 || *in.Rating > 5) {
		return ProductComment{}, fmt.Errorf("%w: rating must be 1-5", ErrInvalidCommentInput)
	}
	id, err := randomID()
	if err != nil {
		return ProductComment{}, err
	}
	now := time.Now().Unix()
	c := ProductComment{
		ID:          id,
		ProductID:   productID,
		Nickname:    nickname,
		Content:     content,
		Rating:      in.Rating,
		Status:      "pending",
		CreatedUnix: now,
		UpdatedUnix: now,
	}
	if err := s.store.InsertComment(ctx, c); err != nil {
		return ProductComment{}, err
	}
	return c, nil
}

// ListComments returns comments for admin moderation, optionally filtered
// by status.
func (s Service) ListComments(ctx context.Context, principal auth.Principal, status string) ([]ProductComment, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return nil, ErrForbidden
	}
	if status != "" && !validCommentStatuses[status] {
		return nil, fmt.Errorf("%w: invalid comment status %q", ErrInvalidAdminInput, status)
	}
	return s.store.ListComments(ctx, status)
}

// ModerateComment transitions a comment to approved or rejected and
// optionally records an admin reply. Pending comments cannot be replied
// to without a status decision — the reply is stored with the moderation
// action so the audit stays atomic.
func (s Service) ModerateComment(ctx context.Context, principal auth.Principal, id, status, reply string) (ProductComment, error) {
	if !auth.Can(principal, "twcommerce.admin") {
		return ProductComment{}, ErrForbidden
	}
	if status != "approved" && status != "rejected" {
		return ProductComment{}, fmt.Errorf("%w: moderation status must be approved or rejected", ErrInvalidAdminInput)
	}
	existing, err := s.store.GetComment(ctx, id)
	if err != nil {
		return ProductComment{}, err
	}
	reply = strings.TrimSpace(reply)
	now := time.Now().Unix()
	repliedUnix := existing.RepliedUnix
	if reply != "" && reply != existing.AdminReply {
		repliedUnix = now
	}
	if err := s.store.UpdateCommentModeration(ctx, id, status, reply, repliedUnix, now); err != nil {
		return ProductComment{}, err
	}
	return s.store.GetComment(ctx, id)
}
