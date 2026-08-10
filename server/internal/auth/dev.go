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
	return Principal{UserID: "local-admin", Email: "local@example.test", Role: "admin"}, nil
}
