package auth

import "context"

type DevVerifier struct {
	token string
}

func NewDevVerifier(token string) DevVerifier {
	return DevVerifier{token: token}
}

func (v DevVerifier) Verify(_ context.Context, token string) (Principal, error) {
	if token == "" || token != v.token {
		return Principal{}, ErrUnauthorized
	}
	return Principal{
		UserID: "local-admin",
		Email:  "local@example.test",
		Role:   "admin",
		Capabilities: []string{
			"twcommerce.read",
			"twcommerce.create",
			"twcommerce.update",
			"twcommerce.delete",
			"twcommerce.admin",
			"orders.returns",
			"inventory.adjust",
			"content.read",
			"content.create",
			"content.update",
			"content.approve",
			"content.publish",
			"staff.read",
			"staff.update",
			"media.upload",
		},
	}, nil
}
