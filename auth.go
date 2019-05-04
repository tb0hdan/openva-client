package main // nolint illegal character

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

type Authenticator struct {
	authToken string
}

func (a *Authenticator) AddAuthToContext(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", a.authToken)
}

func (a *Authenticator) AuthWithTimeout(timeoutSeconds int) (ctx context.Context, cancelFunc context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	ctx = a.AddAuthToContext(ctx)
	return ctx, cancel
}

func (a *Authenticator) AuthWithoutTimeout() (ctx context.Context) {
	return a.AddAuthToContext(context.Background())
}

func (a *Authenticator) AuthWithDeadline(deadlineSeconds int) context.Context {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(deadlineSeconds)*time.Second))
	return a.AddAuthToContext(ctx)
}

func NewAuthenticatorFromString(token string) (*Authenticator, error) {
	authenticator := &Authenticator{}
	return AuthenticatorFromString(authenticator, token)
}

func AuthenticatorFromString(authenticator *Authenticator, token string) (*Authenticator, error) {
	if len(token) == 0 {
		return nil, errors.New("token length is zero")
	}
	authenticator.authToken = token
	return authenticator, nil
}
