// Package auth protects the whole site behind a single OIDC/SSO login.
// There are no local accounts and no per-user permissions: anyone who
// completes the SSO login gets full access, matching the app's existing
// "everyone can access everything" model.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

const (
	sessionCookie = "printcost_session"
	stateCookie   = "printcost_oauth_state"
	sessionTTL    = 30 * 24 * time.Hour
)

// Config holds the OIDC settings, all supplied via environment variables.
type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SessionKey   []byte
	// BasePrefix mirrors main's BASE_URL_PREFIX, so the auth routes and
	// post-login redirect land under the same path the app is served from.
	BasePrefix string
}

type Auth struct {
	cfg      Config
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
}

// New sets up the OIDC provider. It talks to the issuer's discovery
// endpoint, so it can fail if the issuer is unreachable or misconfigured -
// that's treated as fatal by the caller since the whole site depends on it.
func New(ctx context.Context, cfg Config) (*Auth, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery failed: %w", err)
	}

	a := &Auth{
		cfg:      cfg,
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		oauth: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
	}
	return a, nil
}

// Mount wires the unprotected login/callback/logout endpoints under
// "<BasePrefix>/auth".
func (a *Auth) Mount(r chi.Router) {
	r.Get(a.cfg.BasePrefix+"/auth/login", a.handleLogin)
	r.Get(a.cfg.BasePrefix+"/auth/callback", a.handleCallback)
	r.Get(a.cfg.BasePrefix+"/auth/logout", a.handleLogout)
}

func (a *Auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomString()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   int(10 * time.Minute / time.Second),
		HttpOnly: true,
		Secure:   isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, a.oauth.AuthCodeURL(state), http.StatusFound)
}

func (a *Auth) loginPath() string { return a.cfg.BasePrefix + "/auth/login" }
func (a *Auth) homePath() string  { return a.cfg.BasePrefix + "/" }

func (a *Auth) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCk, err := r.Cookie(stateCookie)
	if err != nil || stateCk.Value == "" || stateCk.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}
	clearCookie(w, stateCookie, r)

	token, err := a.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in response", http.StatusBadGateway)
		return
	}
	if _, err := a.verifier.Verify(r.Context(), rawIDToken); err != nil {
		http.Error(w, "id_token verification failed", http.StatusUnauthorized)
		return
	}

	sessionValue, err := a.signSession(time.Now().Add(sessionTTL))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sessionValue,
		Path:     "/",
		MaxAge:   int(sessionTTL / time.Second),
		HttpOnly: true,
		Secure:   isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, a.homePath(), http.StatusFound)
}

func (a *Auth) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, sessionCookie, r)
	http.Redirect(w, r, a.homePath(), http.StatusFound)
}

// Require protects every request behind a valid session: browser
// navigations are redirected to the login flow, API/XHR requests get a
// plain 401 so the frontend can react instead of receiving an HTML page.
func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, a.cfg.BasePrefix+"/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		if a.validSession(r) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, a.cfg.BasePrefix+"/api/") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, a.loginPath(), http.StatusFound)
	})
}

func (a *Auth) validSession(r *http.Request) bool {
	ck, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	exp, ok := a.verifySession(ck.Value)
	return ok && time.Now().Before(exp)
}

// session is the signed cookie payload. There's no user identity beyond
// "authenticated", since every logged-in user has the same full access.
type session struct {
	Expires int64 `json:"exp"`
}

func (a *Auth) signSession(expires time.Time) (string, error) {
	payload, err := json.Marshal(session{Expires: expires.Unix()})
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, a.cfg.SessionKey)
	mac.Write([]byte(encoded))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + sig, nil
}

func (a *Auth) verifySession(value string) (time.Time, bool) {
	encoded, sig, ok := strings.Cut(value, ".")
	if !ok {
		return time.Time{}, false
	}
	mac := hmac.New(sha256.New, a.cfg.SessionKey)
	mac.Write([]byte(encoded))
	wantSig, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(wantSig, mac.Sum(nil)) {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return time.Time{}, false
	}
	var s session
	if err := json.Unmarshal(payload, &s); err != nil {
		return time.Time{}, false
	}
	return time.Unix(s.Expires, 0), true
}

func randomString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func clearCookie(w http.ResponseWriter, name string, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func isSecure(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}
