package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/credentials/insecure"

	// This controls the maxprocs environment variable in container runtimes.
	// see https://martin.baillie.id/wrote/gotchas-in-the-go-network-packages-defaults/#bonus-gomaxprocs-containers-and-the-cfs
	"go.uber.org/automaxprocs/maxprocs"
	"google.golang.org/grpc"

	"github.com/cfhn/our-space/ourspace-backend/internal/cards"
	"github.com/cfhn/our-space/ourspace-backend/internal/config"
	"github.com/cfhn/our-space/ourspace-backend/internal/members"
	"github.com/cfhn/our-space/ourspace-backend/pb"
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

	_, err := maxprocs.Set(maxprocs.Logger(func(s string, i ...interface{}) {
		logger.DebugContext(ctx, fmt.Sprintf(s, i...))
	}))
	if err != nil {
		return fmt.Errorf("setting max procs: %w", err)
	}

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

			err := pb.RegisterMemberServiceHandlerClient(context.Background(), mux, pb.NewMemberServiceClient(client))
			if err != nil {
				return err
			}

			err = pb.RegisterCardServiceHandlerClient(context.Background(), mux, pb.NewCardServiceClient(client))
			if err != nil {
				return err
			}

			return nil
		},
		Jobs: nil,
	}

	return server.Run()
}
