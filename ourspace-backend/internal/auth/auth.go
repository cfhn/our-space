package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/pwhash"
	"github.com/cfhn/our-space/pkg/setup"
	"github.com/cfhn/our-space/pkg/status"
)

const (
	accessTokenValidity  = 15 * time.Minute
	refreshTokenValidity = 8 * time.Hour
	maxSessionLifetime   = 14 * 24 * time.Hour
)

type Repository interface {
	FindUserLoginDetails(ctx context.Context, username string) (*LoginDetails, error)
	UpdateHash(ctx context.Context, username, password string) error
}

type LoginDetails struct {
	ID           string
	Username     string
	PasswordHash string

	FullName string
}

type Service struct {
	pb.UnimplementedAuthServiceServer

	repo       Repository
	issuer     string
	signingKey *atomic.Pointer[ecdsa.PrivateKey]
	publicKeys *atomic.Pointer[map[string]*ecdsa.PublicKey]
}

func NewAuthService(
	repo Repository, signingKey *atomic.Pointer[ecdsa.PrivateKey],
	publicKeys *atomic.Pointer[map[string]*ecdsa.PublicKey],
) *Service {
	return &Service{
		repo:       repo,
		signingKey: signingKey,
		publicKeys: publicKeys,
	}
}

func (s *Service) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	switch c := request.Credentials.(type) {
	case *pb.LoginRequest_Password:
		return s.passwordLogin(ctx, c.Password)
	case *pb.LoginRequest_Oidc:
		return nil, status.Unimplemented()
	default:
		return nil, status.FieldViolations([]*errdetails.BadRequest_FieldViolation{
			{
				Field:       "credentials",
				Description: "invalid credentials type",
			},
		})
	}
}

func (s *Service) passwordLogin(ctx context.Context, credentials *pb.LoginPassword) (*pb.LoginResponse, error) {
	loginDetails, err := s.repo.FindUserLoginDetails(ctx, credentials.Username)
	if errors.Is(err, ErrNotFound) {
		return nil, status.Unauthenticated()
	}
	if err != nil {
		return nil, err
	}

	updatedHash, same := pwhash.Verify(credentials.Password, loginDetails.PasswordHash)
	if !same {
		return nil, status.Unauthenticated()
	}

	if updatedHash != "" {
		err = s.repo.UpdateHash(ctx, credentials.Username, updatedHash)
		if err != nil {
			return nil, status.Internal(err)
		}
	}

	accessToken, accessTokenExpiry, refreshToken, refreshTokenExpiry, err := s.generateTokens(credentials.Username, loginDetails.FullName, time.Now())
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{
		Outcome: &pb.LoginResponse_Success{
			Success: &pb.LoginSuccess{
				AccessToken:        accessToken,
				AccessTokenExpiry:  timestamppb.New(accessTokenExpiry),
				RefreshToken:       refreshToken,
				RefreshTokenExpiry: timestamppb.New(refreshTokenExpiry),
			},
		},
	}, nil
}

func (s *Service) Refresh(ctx context.Context, request *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	refreshTokens := metadata.ValueFromIncomingContext(ctx, "x-refresh-token")
	if len(refreshTokens) != 1 {
		return nil, status.PermissionDenied()
	}

	var refreshTokenClaims setup.RefreshTokenClaims
	_, err := jwt.ParseWithClaims(refreshTokens[0], &refreshTokenClaims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, nil
		}

		keys := s.publicKeys.Load()
		return (*keys)[kid], nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodES256.Name}))
	if err != nil {
		return nil, status.PermissionDenied()
	}

	if refreshTokenClaims.Type != "refresh" {
		return nil, status.PermissionDenied()
	}

	loginDetail, err := s.repo.FindUserLoginDetails(ctx, refreshTokenClaims.Subject)
	if err != nil {
		return nil, status.Internal(err)
	}

	accessToken, accessTokenExpiry, refreshToken, refreshTokenExpiry, err := s.generateTokens(refreshTokenClaims.Subject, loginDetail.FullName, refreshTokenClaims.LoginTime.Time)
	if errors.Is(err, ErrSessionExceedsLifetime) {
		return nil, status.Unauthenticated()
	}
	if err != nil {
		return nil, err
	}

	return &pb.RefreshResponse{
		Success: &pb.LoginSuccess{
			AccessToken:        accessToken,
			RefreshToken:       refreshToken,
			AccessTokenExpiry:  timestamppb.New(accessTokenExpiry),
			RefreshTokenExpiry: timestamppb.New(refreshTokenExpiry),
		},
	}, nil
}

func verify(password, hash string) bool {
	parts := strings.Split(hash, "$")

	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" {
		return false
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false
	}

	if version != argon2.Version {
		return false
	}

	var (
		memory, iteration uint32
		parallelism       uint8
	)
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iteration, &parallelism)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false
	}

	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false
	}

	keyLength := uint32(len(key))

	otherKey := argon2.IDKey([]byte(password), salt, iteration, memory, parallelism, keyLength)
	otherKeyLength := int32(len(key))

	if subtle.ConstantTimeEq(int32(keyLength), otherKeyLength) == 0 {
		return false
	}

	if subtle.ConstantTimeCompare(key, otherKey) == 1 {
		return true
	}

	return false
}

func CookieRewriter(ctx context.Context, w http.ResponseWriter, m proto.Message) error {
	var (
		refreshToken string
		expiry       time.Time
	)

	var loginSuccess *pb.LoginSuccess

	switch response := m.(type) {
	case *pb.LoginResponse:
		loginSuccessResponse, ok := response.Outcome.(*pb.LoginResponse_Success)
		if !ok {
			return nil
		}

		loginSuccess = loginSuccessResponse.Success
	case *pb.RefreshResponse:
		loginSuccess = response.Success
	case *pb.LogoutResponse:
		// Remove refresh token
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh-token",
			Path:     "/api/v1/auth/refresh",
			MaxAge:   -1,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		return nil
	default:
		return nil
	}

	refreshToken = loginSuccess.RefreshToken
	expiry = loginSuccess.RefreshTokenExpiry.AsTime()

	loginSuccess.RefreshToken = ""
	loginSuccess.RefreshTokenExpiry = nil

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh-token",
		Value:    refreshToken,
		Quoted:   false,
		Path:     "/api/v1/auth/refresh",
		Expires:  expiry,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func CookieForwarder(_ context.Context, request *http.Request) metadata.MD {
	cookie, err := request.Cookie("refresh-token")
	if err != nil {
		return nil
	}

	return metadata.Pairs("x-refresh-token", cookie.Value)
}
