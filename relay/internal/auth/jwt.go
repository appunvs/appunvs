// Package auth handles JWT RS256 signing/verification and the /auth/register
// handler.  Keys are loaded from disk; if unavailable an ephemeral pair is
// generated at startup (dev convenience).
package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// Claims is the JWT payload the relay both issues and verifies.
type Claims struct {
	UserID   string `json:"uid"`
	DeviceID string `json:"did,omitempty"`
	Platform string `json:"plat,omitempty"`
	// Typ distinguishes session tokens ("session", dashboard API) from
	// device tokens ("device", WebSocket). A session token has no DeviceID.
	Typ string `json:"typ,omitempty"`
	jwt.RegisteredClaims
}

// Token flavors.
const (
	TokenSession = "session"
	TokenDevice  = "device"
)

// Signer issues and verifies RS256 tokens.
type Signer struct {
	priv        *rsa.PrivateKey
	pub         *rsa.PublicKey
	issuer      string
	audience    string
	sessionTTL  time.Duration
	deviceTTL   time.Duration
}

// NewSigner loads keys from the given paths.  When either path is empty or
// the file is missing, a fresh 2048-bit RSA key pair is generated in memory
// and a warning is logged.  This is sufficient for local dev but must not be
// used in production (tokens die with the process).
//
// sessionTTL governs /auth/signup and /auth/login tokens (short-lived, used
// for dashboard API). deviceTTL governs /auth/register tokens (long-lived,
// used for WebSocket /ws). Pass 0 for deviceTTL to default to 30 days.
func NewSigner(privPath, pubPath, issuer, audience string, sessionTTL, deviceTTL time.Duration, log *zap.Logger) (*Signer, error) {
	if deviceTTL == 0 {
		deviceTTL = 30 * 24 * time.Hour
	}
	if sessionTTL == 0 {
		sessionTTL = 24 * time.Hour
	}
	s := &Signer{issuer: issuer, audience: audience, sessionTTL: sessionTTL, deviceTTL: deviceTTL}

	priv, err := loadPrivateKey(privPath)
	if err != nil {
		log.Warn("auth: generating ephemeral RS256 keypair; tokens will not survive restart",
			zap.String("reason", err.Error()))
		gen, genErr := rsa.GenerateKey(rand.Reader, 2048)
		if genErr != nil {
			return nil, fmt.Errorf("auth: generate ephemeral key: %w", genErr)
		}
		s.priv = gen
		s.pub = &gen.PublicKey
		return s, nil
	}
	s.priv = priv

	pub, err := loadPublicKey(pubPath)
	if err != nil {
		// Fall back to the public half of the private key.
		log.Warn("auth: public key unreadable; deriving from private key",
			zap.String("path", pubPath), zap.Error(err))
		s.pub = &priv.PublicKey
	} else {
		s.pub = pub
	}
	return s, nil
}

// IssueSession mints a session JWT for dashboard API use.
// Session tokens have no DeviceID.
func (s *Signer) IssueSession(userID string) (string, error) {
	return s.issue(userID, "", "", TokenSession, s.sessionTTL)
}

// IssueDevice mints a device JWT for WebSocket use.
func (s *Signer) IssueDevice(userID, deviceID, platform string) (string, error) {
	return s.issue(userID, deviceID, platform, TokenDevice, s.deviceTTL)
}

// Issue is the legacy single-flavor entrypoint. Present-tense callers should
// use IssueSession or IssueDevice; the integration tests still call Issue to
// produce a device token for /ws. Kept for backwards compatibility.
func (s *Signer) Issue(userID, deviceID, platform string) (string, error) {
	return s.IssueDevice(userID, deviceID, platform)
}

func (s *Signer) issue(userID, deviceID, platform, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		DeviceID: deviceID,
		Platform: platform,
		Typ:      typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{s.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return tok.SignedString(s.priv)
}

// Verify parses and validates a token string, returning its claims.
// wantTyp filters by flavor: pass TokenSession, TokenDevice, or "" for any.
// An empty Typ claim on an otherwise valid token is treated as TokenDevice
// for backwards compatibility with legacy scaffold tokens.
func (s *Signer) Verify(tokenStr string, wantTyp string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method %v", t.Header["alg"])
		}
		return s.pub, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithAudience(s.audience))
	if err != nil {
		return nil, err
	}
	if claims.UserID == "" {
		return nil, errors.New("auth: token missing uid")
	}
	typ := claims.Typ
	if typ == "" {
		typ = TokenDevice
	}
	switch wantTyp {
	case "":
		// caller doesn't care
	case TokenSession:
		if typ != TokenSession {
			return nil, errors.New("auth: session token required")
		}
	case TokenDevice:
		if typ != TokenDevice {
			return nil, errors.New("auth: device token required")
		}
		if claims.DeviceID == "" {
			return nil, errors.New("auth: device token missing did")
		}
	default:
		return nil, fmt.Errorf("auth: unknown want typ %q", wantTyp)
	}
	return claims, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, errors.New("no private_key_path configured")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("auth: %s is not PEM", path)
	}
	// Try PKCS1 first, then PKCS8.
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	any, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("auth: parse private key: %w", err)
	}
	k, ok := any.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("auth: private key is not RSA")
	}
	return k, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	if path == "" {
		return nil, errors.New("no public_key_path configured")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("auth: %s is not PEM", path)
	}
	if k, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return k, nil
	}
	any, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("auth: parse public key: %w", err)
	}
	k, ok := any.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("auth: public key is not RSA")
	}
	return k, nil
}
