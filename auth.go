package main

import (
	"context"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

type Authenticator struct {
	authFileName string
	authToken string
}

func (a *Authenticator) ReadAuthData() (token string, err error){
	data, err := ioutil.ReadFile(a.authFileName)
	if err != nil {
		return "", errors.Wrap(err, "ReadAuthData failed")
	}
	return strings.Trim(string(data), " \n"), nil
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

func NewAuthenticator(authFileName string) (*Authenticator, error) {
	authenticator := &Authenticator{
		authFileName: authFileName,
	}
	token, err := authenticator.ReadAuthData()
	if err != nil {
		return nil, err
	}
	if len(token) == 0 {
		return nil, errors.New("Token length is zero")
	}
	authenticator.authToken = token
	return authenticator, nil
}
