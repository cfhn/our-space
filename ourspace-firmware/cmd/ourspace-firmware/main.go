package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/cfhn/our-space/ourspace-firmware/internal/firmware"
	"github.com/cfhn/our-space/ourspace-firmware/internal/frontend"
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

	dataDirectory := os.Getenv("DATA_DIRECTORY")
	repo := inmemory.NewRepository()
	firmwareService := firmware.NewService(logger, repo, dataDirectory)

	synchronizer := &sync.BackendSynchronizer{
		Repository: repo,
		Logger:     logger.With("module", "sync"),
	}

	reloadBackendConnection(dataDirectory, &synchronizer.BackendConn, &synchronizer.APIKey, logger)

	frontendServer := http.Server{
		Handler:     frontend.ServeFrontend(),
		Addr:        ":8080",
		ReadTimeout: 10 * time.Second,
	}

	server := setup.Server{
		HTTPPort: 8081,
		GRPCPort: 8082,
		Logger:   logger,
		Register: func(server *grpc.Server, conn *grpc.ClientConn, mux *runtime.ServeMux) error {
			pb.RegisterFirmwareServiceServer(server, firmwareService)

			return errors.Join(
				pb.RegisterFirmwareServiceHandler(context.Background(), mux, conn),
				mux.HandlePath(http.MethodGet, "/card-events", sse.GrpcProxy[*pb.ListenForCardEventsRequest, *pb.ListenForCardEventsResponse](conn, pb.FirmwareService_ServiceDesc.Streams[0], pb.FirmwareService_ListenForCardEvents_FullMethodName)),
			)
		},
		Jobs: []setup.JobSpec{
			{
				Name:      "Synchronize",
				Job:       setup.JobFunc(synchronizer.Synchronize),
				Interval:  10 * time.Second,
				Immediate: true,
			},
			{
				Name: "Frontend",
				Job: setup.JobFunc(func(ctx context.Context) error {
					err := frontendServer.ListenAndServe()
					if errors.Is(err, http.ErrServerClosed) {
						return nil
					}

					return err
				}),
				Interval:  math.MaxInt,
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

func loadConfig(dataDirectory string) *pb.Configuration {
	f, err := os.Open(filepath.Join(dataDirectory, "config.json"))
	if err != nil {
		return nil
	}

	defer f.Close()

	configBytes, err := io.ReadAll(f)
	if err != nil {
		return nil
	}

	config := &pb.Configuration{}
	if err := protojson.Unmarshal(configBytes, config); err != nil {
		return nil
	}

	return config
}

func reloadBackendConnection(
	dataDirectory string, backendConn *atomic.Pointer[grpc.ClientConn], apiKey *atomic.Pointer[string],
	logger *slog.Logger,
) {
	config := loadConfig(dataDirectory)
	if config == nil {
		return
	}

	var creds credentials.TransportCredentials
	if config.BackendTls {
		creds = credentials.NewTLS(&tls.Config{})
	} else {
		creds = insecure.NewCredentials()
	}

	clientConn, err := grpc.NewClient(config.BackendUrl, grpc.WithTransportCredentials(creds))
	if err != nil {
		logger.Error("failed to connect to backend", slog.Any("error", err))

		return
	}

	oldConn := backendConn.Swap(clientConn)
	apiKey.Store(new(config.ApiKey))

	err = oldConn.Close()
	if err != nil {
		logger.Warn("error closing old connection", slog.Any("error", err))
	}
}
