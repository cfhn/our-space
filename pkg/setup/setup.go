package setup

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/cfhn/our-space/pkg/log"
)

type JobSpec struct {
	Name      string
	Job       Job
	Interval  time.Duration
	Immediate bool
}

type Job interface {
	Run(ctx context.Context) error
}

type JobFunc func(ctx context.Context) error

func (fn JobFunc) Run(ctx context.Context) error {
	return fn(ctx)
}

type Server struct {
	HTTPPort        int
	GRPCPort        int
	Logger          *slog.Logger
	Register        func(*grpc.Server, *grpc.ClientConn, *runtime.ServeMux) error
	Jobs            []JobSpec
	Cors            *cors.Options
	ServeMuxOptions []runtime.ServeMuxOption

	DisableAuthentication bool
	KeyFunc               func(kid string) *ecdsa.PublicKey
}

func (s *Server) Run() error {
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.GRPCPort))
	if err != nil {
		return err
	}

	httpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.HTTPPort))
	if err != nil {
		return err
	}

	muxOptions := s.ServeMuxOptions
	muxOptions = append(muxOptions, runtime.WithIncomingHeaderMatcher(func(s string) (string, bool) {
		return "", strings.EqualFold(s, "authorization")
	}))

	serveMux := runtime.NewServeMux(s.ServeMuxOptions...)

	err = serveMux.HandlePath(
		http.MethodGet, "/.well-known/ready",
		func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
			w.WriteHeader(http.StatusOK)
		},
	)
	if err != nil {
		return err
	}

	var handler http.Handler = serveMux
	if s.Cors != nil {
		handler = cors.New(*s.Cors).Handler(handler)
	}

	httpServer := http.Server{
		Handler: handler,
	}

	interceptors := []grpc.UnaryServerInterceptor{}

	if !s.DisableAuthentication {
		interceptors = append(interceptors, AuthInterceptor(s.KeyFunc))
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	grpcClient, err := grpc.NewClient(fmt.Sprintf("localhost:%d", s.GRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	err = s.Register(server, grpcClient, serveMux)
	if err != nil {
		return err
	}

	reflection.Register(server)

	eg, ctx := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		s.Logger.Info("serving gRPC", "addr", grpcListener.Addr().String())
		err := server.Serve(grpcListener)
		if err != nil {
			s.Logger.Error("error serving gRPC", log.Error(err))
		}

		return err
	})

	eg.Go(func() error {
		s.Logger.Info("serving http", "addr", httpListener.Addr().String())
		err := httpServer.Serve(httpListener)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		if err != nil {
			s.Logger.Error("error serving http", log.Error(err))
		}

		return err
	})

	for _, jobSpec := range s.Jobs {
		eg.Go(func() error {
			ticker := time.NewTicker(jobSpec.Interval)
			defer ticker.Stop()
			if jobSpec.Immediate {
				err := jobSpec.Job.Run(context.Background())
				if err != nil {
					s.Logger.Error("job failure", log.Error(err), slog.String("job", jobSpec.Name))
				}
			}

			for {
				select {
				case <-ticker.C:
					err := jobSpec.Job.Run(context.Background())
					if err != nil {
						s.Logger.Error("job failure", log.Error(err), slog.String("job", jobSpec.Name))
					}
				case <-ctx.Done():
					return nil
				}
			}
		})
	}

	return eg.Wait()
}
