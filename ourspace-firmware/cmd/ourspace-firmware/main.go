package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/ourspace-firmware/internal/firmware"
	"github.com/cfhn/our-space/ourspace-firmware/internal/inmemory"
	"github.com/cfhn/our-space/ourspace-firmware/internal/sync"
	pb "github.com/cfhn/our-space/ourspace-firmware/proto"
	"github.com/cfhn/our-space/pkg/log"
	"github.com/cfhn/our-space/pkg/setup"
	"github.com/cfhn/our-space/pkg/sse"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
	}
}

func run() error {
	// Logger configuration
	logger := log.New(
		log.WithLevel(os.Getenv("LOG_LEVEL")),
		log.WithSource(),
	)

	backendAddress := os.Getenv("BACKEND_ADDRESS")
	backendClient, err := grpc.NewClient(backendAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	repo := inmemory.NewRepository()
	firmwareService := firmware.NewService(logger, repo)

	synchronizer := &sync.BackendSynchronizer{
		AuthClient:   pbBackend.NewAuthServiceClient(backendClient),
		MemberClient: pbBackend.NewMemberServiceClient(backendClient),
		CardClient:   pbBackend.NewCardServiceClient(backendClient),
		Repository:   repo,
		Logger:       logger.With("module", "sync"),

		ApiKey: os.Getenv("API_KEY"),
	}

	server := setup.Server{
		HTTPPort: 8081,
		GRPCPort: 8082,
		Logger:   logger,
		Register: func(server *grpc.Server, conn *grpc.ClientConn, mux *runtime.ServeMux) error {
			pb.RegisterFirmwareServiceServer(server, firmwareService)

			err := pb.RegisterFirmwareServiceHandler(context.Background(), mux, conn)
			if err != nil {
				return err
			}
			err = mux.HandlePath(http.MethodGet, "/card-events", sse.GrpcProxy[*pb.ListenForCardEventsRequest, *pb.ListenForCardEventsResponse](conn, pb.FirmwareService_ServiceDesc.Streams[0], pb.FirmwareService_ListenForCardEvents_FullMethodName))
			if err != nil {
				return err
			}
			return nil
		},
		Jobs: []setup.JobSpec{
			{
				Name:      "Synchronize",
				Job:       setup.JobFunc(synchronizer.Synchronize),
				Interval:  10 * time.Second,
				Immediate: true,
			},
		},
		Cors: &cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{
				http.MethodGet, http.MethodPost, http.MethodPatch,
				http.MethodPut, http.MethodDelete,
			},
			AllowedHeaders:   nil,
			ExposedHeaders:   nil,
			MaxAge:           0,
			AllowCredentials: true,
		},
		DisableAuthentication: true,
	}

	return server.Run()
}
