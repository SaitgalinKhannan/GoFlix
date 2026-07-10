// Package auth handles JWT issuance and verification for the API.
// It implements an access/refresh token pair: a short-lived access token
// authorizes API requests, a long-lived refresh token obtains new access tokens.
package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Access tokens authorize requests and expire quickly.
	AccessTokenTTL = 15 * time.Minute
	// Refresh tokens obtain new access tokens and live much longer.
	RefreshTokenTTL = 30 * 24 * time.Hour // 30 days
)

// Token kinds.
const (
	accessTokenKind  = "access"
	refreshTokenKind = "refresh"
)

// AccessTokenKind returns the access-token kind, for the middleware.
func AccessTokenKind() string { return accessTokenKind }

// RefreshTokenKind returns the refresh-token kind.
func RefreshTokenKind() string { return refreshTokenKind }

// Claims are the JWT claims carried by both access and refresh tokens.
type Claims struct {
	Username string `json:"username"`
	Kind     string `json:"kind"`
	jwt.RegisteredClaims
}

// Service issues and verifies JWTs for a single configured user.
type Service struct {
	username     string
	passwordHash []byte
	secret       []byte
	issuer       string
}

// NewService creates an auth service for the given user. passwordHash is a
// bcrypt hash. secret signs the tokens; issuer is recorded in the `iss` claim.
func NewService(username, passwordHash string, secret, issuer []byte) *Service {
	return &Service{
		username:     username,
		passwordHash: []byte(passwordHash),
		secret:       secret,
		issuer:       string(issuer),
	}
}

// HashPassword returns a bcrypt hash of the password (used at provisioning time).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyCredentials checks a username/password pair against the configured user.
func (s *Service) VerifyCredentials(username, password string) error {
	if username != s.username {
		return errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword(s.passwordHash, []byte(password)); err != nil {
		return errors.New("invalid credentials")
	}
	return nil
}

// TokenPair holds a freshly issued access/refresh pair.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // seconds until the access token expires
}

// Issue issues a new access/refresh pair for the configured user.
func (s *Service) Issue() (TokenPair, error) {
	access, err := s.sign(s.username, accessTokenKind, AccessTokenTTL)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := s.sign(s.username, refreshTokenKind, RefreshTokenTTL)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(AccessTokenTTL.Seconds()),
	}, nil
}

func (s *Service) sign(username, kind string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		Kind:     kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// Verify parses and validates a token of the expected kind.
func (s *Service) Verify(tokenStr, expectedKind string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	if expectedKind != "" && claims.Kind != expectedKind {
		return nil, fmt.Errorf("wrong token kind: want %q, got %q", expectedKind, claims.Kind)
	}
	return claims, nil
}

// ExtractBearer pulls the token out of an Authorization: Bearer <token> header.
func ExtractBearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}
