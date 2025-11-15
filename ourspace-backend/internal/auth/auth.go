package auth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	accessTokenValidity       = 15 * time.Minute
	apiKeyAccessTokenValidity = 1 * time.Hour
	refreshTokenValidity      = 8 * time.Hour
	maxSessionLifetime        = 14 * 24 * time.Hour
)

type Repository interface {
	FindUserLoginDetails(ctx context.Context, username string) (*LoginDetails, error)
	FindApiKey(ctx context.Context, apiKey string) (*ApiKeyDetails, error)
	UpdateHash(ctx context.Context, username, password string) error
}

type LoginDetails struct {
	ID           string
	Username     string
	PasswordHash string

	FullName string
}

type ApiKeyDetails struct {
	ID       string
	MemberId string
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
	case *pb.LoginRequest_ApiKey:
		return s.apiKeyLogin(ctx, c.ApiKey)
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
	if errors.Is(err, ErrUserNotFound) {
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

	accessToken, accessTokenExpiry, refreshToken, refreshTokenExpiry, err := s.generateTokens(
		credentials.Username,
		loginDetails.FullName,
		time.Now(),
		accessTokenValidity,
	)
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

func (s *Service) apiKeyLogin(ctx context.Context, credentials *pb.LoginApiKey) (*pb.LoginResponse, error) {
	apiKeyDetails, err := s.repo.FindApiKey(ctx, credentials.ApiKey)
	if err != nil {
		return nil, err
	}

	if errors.Is(err, ErrApiKeyNotFound) {
		return nil, status.Unauthenticated()
	}
	if err != nil {
		return nil, status.Internal(err)
	}

	accessToken, accessTokenExpiry, _, _, err := s.generateTokens(
		apiKeyDetails.ID,
		"",
		time.Now(),
		apiKeyAccessTokenValidity,
	)

	return &pb.LoginResponse{
		Outcome: &pb.LoginResponse_Success{
			Success: &pb.LoginSuccess{
				AccessToken:       accessToken,
				AccessTokenExpiry: timestamppb.New(accessTokenExpiry),
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

	accessToken, accessTokenExpiry, refreshToken, refreshTokenExpiry, err := s.generateTokens(
		refreshTokenClaims.Subject,
		loginDetail.FullName,
		refreshTokenClaims.LoginTime.Time,
		accessTokenValidity,
	)
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

func CookieRewriter(_ context.Context, w http.ResponseWriter, m proto.Message) error {
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
	if loginSuccess.RefreshTokenExpiry != nil {
		expiry = loginSuccess.RefreshTokenExpiry.AsTime()
	}

	loginSuccess.RefreshToken = ""
	loginSuccess.RefreshTokenExpiry = nil

	if refreshToken != "" {
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
	}

	return nil
}

func CookieForwarder(_ context.Context, request *http.Request) metadata.MD {
	cookie, err := request.Cookie("refresh-token")
	if err != nil {
		return nil
	}

	return metadata.Pairs("x-refresh-token", cookie.Value)
}
