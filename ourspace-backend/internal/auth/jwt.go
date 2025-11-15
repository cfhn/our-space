package auth

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/cfhn/our-space/pkg/setup"
)

var ErrSessionExceedsLifetime = errors.New("session max length exceeds lifetime")

func (s *Service) generateTokens(
	username, name string, loginTime time.Time, accessTokenValidity time.Duration,
) (string, time.Time, string, time.Time, error) {
	now := time.Now()
	accessToken := jwt.NewWithClaims(jwt.SigningMethodES256, setup.AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   username,
			Audience:  nil,
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenValidity)),
			NotBefore: jwt.NewNumericDate(now.Add(-15 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
		Type:     "access",
		FullName: name,
	})

	signignKey := s.signingKey.Load()
	kid, err := PublicKeyFingerprint(&signignKey.PublicKey)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	accessToken.Header["kid"] = kid

	signedAccessToken, err := accessToken.SignedString(s.signingKey.Load())
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	refreshTokenExpiry := timeMin(loginTime.Add(maxSessionLifetime), now.Add(refreshTokenValidity))
	if refreshTokenExpiry.Before(now) {
		return "", time.Time{}, "", time.Time{}, ErrSessionExceedsLifetime
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodES256, setup.RefreshTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   username,
			Audience:  nil,
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenValidity)),
			NotBefore: jwt.NewNumericDate(now.Add(-15 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
		Type:      "refresh",
		LoginTime: jwt.NewNumericDate(loginTime),
	})

	refreshToken.Header["kid"] = kid

	signedRefreshToken, err := refreshToken.SignedString(s.signingKey.Load())
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	return signedAccessToken, now.Add(accessTokenValidity), signedRefreshToken, now.Add(refreshTokenValidity), nil
}

func timeMin(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func PublicKeyFingerprint(publicKey *ecdsa.PublicKey) (string, error) {
	signingKeyBytes, err := publicKey.Bytes()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(signingKeyBytes)
	kid := base64.RawStdEncoding.EncodeToString(hash[:])
	return kid, nil
}
