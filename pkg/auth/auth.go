package auth

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/browser"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func env(name, dflt string) string {
	v, ok := os.LookupEnv(name)
	if !ok {
		return dflt
	}
	return v
}

var (
	domain     = env("AUTH_DOMAIN", "klotho.us.auth0.com")
	clientId   = env("AUTH_CLIENT_ID", "6KQhBRK03c5FWOiJvVZUsEZSjWJ0dvQ1")
	browserEnv = env("BROWSER", "")

	//go:embed auth0_client_secret.key
	clientSecret string
)

func GetAuthToken(ctx context.Context) (*oauth2.Token, *http.Client, error) {
	log := zap.S().Named("auth")

	auth, err := newAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	if token := readCachedToken(ctx, auth); token != nil {
		return token, auth.HTTPClient(ctx, token), nil
	}

	tokenCh := make(chan *oauth2.Token)

	callbackUrl, err := url.Parse(auth.RedirectURL)
	if err != nil {
		return nil, nil, err
	}

	state, err := generateRandomState()
	if err != nil {
		return nil, nil, err
	}

	srv := &http.Server{Addr: callbackUrl.Host}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("received request: %s %s", r.Method, r.URL.Path)

		if r.URL.Path != "/callback" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if reqState := r.URL.Query().Get("state"); reqState != state {
			log.Warnf("Got mismatched state: expected %s, got %s", state, reqState)
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}
		token, err := auth.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "failed to exchange token", http.StatusInternalServerError)
			log.Errorf("failed to exchange token: %+v", err)
			return
		}
		_, err = auth.VerifyIDToken(r.Context(), token)
		if err != nil {
			http.Error(w, "failed to verify token", http.StatusInternalServerError)
			log.Errorf("failed to verify token: %+v", err)
			return
		}
		token.AccessToken = token.Extra("id_token").(string)
		tokenCh <- token
		fmt.Fprint(w, `<html><body>Success, you can now close this window</body></html>`)
		log.Debugf("successfully authenticated")
	})

	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			zap.S().Errorw("failed to shutdown server", "error", err)
		}
	}()

	ready := make(chan struct{})
	srvErrCh := make(chan error)

	go func() {
		ln, err := net.Listen("tcp", srv.Addr)
		if err != nil {
			srvErrCh <- err
			return
		}
		ready <- struct{}{}
		srvErrCh <- srv.Serve(ln)
	}()

	for {
		select {
		case <-ready:
			dest := auth.AuthCodeURL(state)
			if strings.ToLower(browserEnv) == "none" {
				fmt.Printf("To authenticate, please visit:\n\t%s\n", dest)
				continue
			}
			err := browser.OpenURL(dest)
			if err != nil {
				return nil, nil, err
			}

		case err := <-srvErrCh:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return nil, nil, err
			}

		case token := <-tokenCh:
			writeCachedToken(token)
			return token, auth.HTTPClient(ctx, token), nil
		}
	}
}

func readCachedToken(ctx context.Context, auth *Authenticator) *oauth2.Token {
	log := zap.S().Named("auth.cache")

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Debugf("failed to get cache dir: %v", err)
		return nil
	}
	cacheLoc := filepath.Join(cacheDir, "klotho", "token")
	cacheFile, err := os.Open(cacheLoc)
	if err != nil {
		log.Debugf("failed to open cache file: %v", err)
		return nil
	}
	var token oauth2.Token
	if err := json.NewDecoder(cacheFile).Decode(&token); err == nil {
		oidcConfig := &oidc.Config{ClientID: auth.ClientID}
		// ID token replaces access token, so use that for verification
		idToken, err := auth.Verifier(oidcConfig).Verify(ctx, token.AccessToken)
		if err != nil {
			log.Debugf("failed to verify token: %v", err)
			return nil
		}
		if token.Valid() && idToken.Expiry.After(time.Now()) {
			log.Debugf("using cached token")
			return &token
		}
		if idToken.Issuer != auth.Config.Endpoint.AuthURL {
			log.Debugf("token issuer does not match auth endpoint")
		}
		if token.RefreshToken == "" {
			log.Debugf("token is invalid and has no refresh token")
			return nil
		}
	} else {
		log.Debugf("failed to decode token: %v", err)
	}
	return nil
}

func writeCachedToken(token *oauth2.Token) {
	log := zap.S().Named("auth.cache")

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Debugf("failed to get cache dir: %v", err)
		return
	}
	cacheLoc := filepath.Join(cacheDir, "klotho", "token")
	err = os.MkdirAll(filepath.Dir(cacheLoc), 0700)
	if err != nil {
		log.Debugf("failed to create cache dir: %v", err)
		return
	}
	cacheFile, err := os.OpenFile(cacheLoc, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Debugf("failed to open cache file: %v", err)
		return
	}
	defer cacheFile.Close()
	err = json.NewEncoder(cacheFile).Encode(token)
	if err != nil {
		log.Debugf("failed to write token: %v", err)
	}
}

type Authenticator struct {
	*oidc.Provider
	oauth2.Config
}

func newAuth(ctx context.Context) (*Authenticator, error) {
	provider, err := oidc.NewProvider(
		ctx,
		"https://"+domain+"/",
	)
	if err != nil {
		return nil, err
	}

	if clientSecret == "" {
		return nil, errors.New("missing client secret (pkg/auth/auth0_client_secret.key not embedded)")
	}

	conf := oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:3104/callback",
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	return &Authenticator{
		Provider: provider,
		Config:   conf,
	}, nil
}

// VerifyIDToken verifies that an *oauth2.Token is a valid *oidc.IDToken.
func (a *Authenticator) VerifyIDToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	oidcConfig := &oidc.Config{
		ClientID: a.ClientID,
	}

	return a.Verifier(oidcConfig).Verify(ctx, rawIDToken)
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	state := base64.StdEncoding.EncodeToString(b)

	return state, nil
}

type idTokenSource struct {
	src  oauth2.TokenSource
	auth *Authenticator
}

func (s *idTokenSource) Token() (*oauth2.Token, error) {
	t, err := s.src.Token()
	if err != nil {
		return nil, err
	}
	if id, ok := t.Extra("id_token").(string); ok {
		idToken, err := s.auth.VerifyIDToken(context.Background(), t)
		if err != nil {
			return nil, err
		}

		// per TokenSource contract, we must return a copy if modifying
		tCopy := *t
		t = &tCopy
		t.AccessToken = id
		t.Expiry = idToken.Expiry
	}
	return t, nil
}

func (auth *Authenticator) HTTPClient(ctx context.Context, token *oauth2.Token) *http.Client {
	ts := oauth2.ReuseTokenSource(token, &idTokenSource{src: auth.Config.TokenSource(ctx, token)})
	return oauth2.NewClient(ctx, ts)
}
