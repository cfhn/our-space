package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"

	"github.com/cfhn/our-space/ourspace-backend/internal/auth"
	"github.com/cfhn/our-space/ourspace-backend/internal/cards"
	"github.com/cfhn/our-space/ourspace-backend/internal/config"
	"github.com/cfhn/our-space/ourspace-backend/internal/members"
	"github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/database"
	"github.com/cfhn/our-space/pkg/log"
	"github.com/cfhn/our-space/pkg/setup"
)

func main() {
	// Logger configuration
	logger := log.New(
		log.WithLevel(os.Getenv("LOG_LEVEL")),
		log.WithSource(),
	)

	if err := run(logger); err != nil {
		logger.ErrorContext(context.Background(), "an error occurred", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx := context.Background()

	cfg, err := config.Get()
	if err != nil {
		return err
	}

	db, err := database.Connect(database.Config{
		URI:          cfg.Database.URL,
		MaxOpenConns: cfg.Database.MaxOpenConns,
	})
	if err != nil {
		return err
	}

	err = database.Migrate(ctx, db, logger)
	if err != nil {
		return err
	}

	client, err := grpc.NewClient(fmt.Sprintf("localhost:%d", cfg.GRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	var (
		signingKey atomic.Pointer[ecdsa.PrivateKey]
		publicKeys atomic.Pointer[map[string]*ecdsa.PublicKey]
	)

	authRepo := auth.NewPostgresRepo(db)
	authService := auth.NewAuthService(authRepo, &signingKey, &publicKeys)
	membersRepo := members.NewPostgresRepo(db)
	memberService := members.NewService(membersRepo)
	cardsRepo := cards.NewPostgresRepo(db)
	cardsService := cards.NewService(cardsRepo, memberService)

	server := setup.Server{
		HTTPPort: cfg.HTTPPort,
		GRPCPort: cfg.GRPCPort,
		Logger:   logger,
		Register: func(server *grpc.Server, conn *grpc.ClientConn, mux *runtime.ServeMux) error {
			pb.RegisterMemberServiceServer(server, memberService)
			pb.RegisterCardServiceServer(server, cardsService)
			pb.RegisterAuthServiceServer(server, authService)

			err := pb.RegisterMemberServiceHandlerClient(context.Background(), mux, pb.NewMemberServiceClient(client))
			if err != nil {
				return err
			}

			err = pb.RegisterCardServiceHandlerClient(context.Background(), mux, pb.NewCardServiceClient(client))
			if err != nil {
				return err
			}

			err = pb.RegisterAuthServiceHandlerClient(context.Background(), mux, pb.NewAuthServiceClient(client))
			if err != nil {
				return err
			}

			return nil
		},
		Jobs: []setup.JobSpec{
			{
				Name:      "refresh_signing_key",
				Immediate: true,
				Interval:  5 * time.Minute,
				Job: setup.JobFunc(func(ctx context.Context) error {
					loadedSigningKey, verificationKeys, err := loadKeys(cfg.Auth.SigningKeyPath, cfg.Auth.VerificationKeysPath)
					if err != nil {
						return err
					}

					publicKeys.Store(&verificationKeys)
					signingKey.Store(loadedSigningKey)

					return nil
				}),
			},
		},
		ServeMuxOptions: []runtime.ServeMuxOption{
			runtime.WithForwardResponseOption(auth.CookieRewriter),
			runtime.WithMetadata(auth.CookieForwarder),
		},
		KeyFunc: func(kid string) *ecdsa.PublicKey {
			keyMap := *publicKeys.Load()
			return keyMap[kid]
		},
	}

	return server.Run()
}

func loadKeys(privateKeyPath, publicKeysPath string) (*ecdsa.PrivateKey, map[string]*ecdsa.PublicKey, error) {
	pemContent, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(pemContent)
	if block == nil {
		return nil, nil, fmt.Errorf("public key not found")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	pemContent, err = os.ReadFile(publicKeysPath)
	if err != nil {
		return nil, nil, err
	}

	keys := map[string]*ecdsa.PublicKey{}

	for {
		block, pemContent = pem.Decode(pemContent)
		if block == nil {
			break
		}

		if block.Type != "PUBLIC KEY" {
			continue
		}

		publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, nil, err
		}

		ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			continue
		}

		b, err := ecdsaKey.Bytes()
		if err != nil {
			return nil, nil, err
		}

		fingerprint := sha256.Sum256(b)
		encoded := base64.RawStdEncoding.EncodeToString(fingerprint[:])

		keys[encoded] = ecdsaKey
	}

	// Always add the current signing key to the verification keys
	signingPublicKey := privateKey.Public().(*ecdsa.PublicKey)
	b, err := signingPublicKey.Bytes()
	if err != nil {
		return nil, nil, err
	}

	fingerprint := sha256.Sum256(b)
	encoded := base64.RawStdEncoding.EncodeToString(fingerprint[:])

	keys[encoded] = signingPublicKey

	return privateKey, keys, nil
}
