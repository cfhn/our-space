package sync

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
	pb "github.com/cfhn/our-space/ourspace-firmware/proto"
	"github.com/cfhn/our-space/pkg/setup"
)

var ErrUnknownLoginOutcome = errors.New("unknown login outcome")

type Repository interface {
	Replace(members []*pbBackend.Member, cards []*pbBackend.Card)
	ListPresences() []*pb.LocalPresence
	UpdatePresence(presence *pb.LocalPresence)
}

type BackendSynchronizer struct {
	BackendConn atomic.Pointer[grpc.ClientConn]
	APIKey      atomic.Pointer[string]

	Repository Repository
	Logger     *slog.Logger
}

func (b *BackendSynchronizer) Synchronize(ctx context.Context) error {
	conn := b.BackendConn.Load()
	apiKey := b.APIKey.Load()
	if conn == nil || apiKey == nil {
		b.Logger.InfoContext(ctx, "no backend connection, skipping run")

		return nil
	}

	authClient := pbBackend.NewAuthServiceClient(conn)
	memberClient := pbBackend.NewMemberServiceClient(conn)
	cardClient := pbBackend.NewCardServiceClient(conn)
	presenceClient := pbBackend.NewPresenceServiceClient(conn)

	b.Logger.InfoContext(ctx, "starting sync")

	backendAuth, err := setup.NewBearerTokenAuth(func(ctx context.Context) (string, error) {
		loginResp, err := authClient.Login(ctx, &pbBackend.LoginRequest{
			Credentials: &pbBackend.LoginRequest_ApiKey{
				ApiKey: &pbBackend.LoginApiKey{
					ApiKey: *apiKey,
				},
			},
		})
		if err != nil {
			return "", err
		}

		loginSuccess, ok := loginResp.Outcome.(*pbBackend.LoginResponse_Success)
		if !ok {
			return "", fmt.Errorf("%w: %T", ErrUnknownLoginOutcome, loginResp.Outcome)
		}

		return loginSuccess.Success.AccessToken, nil
	})
	if err != nil {
		return err
	}

	members, err := collect(pageIterator(func(pageToken string) (*pbBackend.ListMembersResponse, error) {
		return memberClient.ListMembers(ctx, &pbBackend.ListMembersRequest{
			PageToken: pageToken,
			PageSize:  100,
		}, grpc.PerRPCCredentials(backendAuth))
	}, (*pbBackend.ListMembersResponse).GetMembers))
	if err != nil {
		return err
	}

	cards, err := collect(pageIterator(func(pageToken string) (*pbBackend.ListCardsResponse, error) {
		return cardClient.ListCards(ctx, &pbBackend.ListCardsRequest{
			PageSize:  100,
			PageToken: pageToken,
		}, grpc.PerRPCCredentials(backendAuth))
	}, (*pbBackend.ListCardsResponse).GetCards))
	if err != nil {
		return err
	}

	b.Repository.Replace(members, cards)

	b.Logger.InfoContext(ctx, "backend->device sync done", slog.Int("members", len(members)), slog.Int("cards", len(cards)))

	var (
		presenceCreated int
		presenceUpdated int
	)

	for _, p := range b.Repository.ListPresences() {
		if p.SynchronizedAt == nil {
			_, err := presenceClient.CreatePresence(ctx, &pbBackend.CreatePresenceRequest{
				PresenceId: p.Presence.Id,
				Presence:   p.Presence,
			}, grpc.PerRPCCredentials(backendAuth))
			if gprcStatus, ok := status.FromError(err); ok && gprcStatus.Code() == codes.AlreadyExists {
				// set SynchronizedAt to a non-nil value, so the presence record is updated next sync run
				p.SynchronizedAt = timestamppb.New(time.Time{})
				p.ModifiedAt = timestamppb.Now()
			}

			if err != nil {
				b.Logger.ErrorContext(ctx, "presence creation error",
					slog.String("presence_id", p.Presence.Id),
					slog.String("error", err.Error()),
				)

				continue
			}

			presenceCreated++
			p.SynchronizedAt = timestamppb.Now()
		} else if p.SynchronizedAt.AsTime().Before(p.ModifiedAt.AsTime()) {
			_, err := presenceClient.UpdatePresence(ctx, &pbBackend.UpdatePresenceRequest{
				Presence:  p.Presence,
				FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"checkin_time", "checkout_time"}},
			}, grpc.PerRPCCredentials(backendAuth))
			if err != nil {
				b.Logger.ErrorContext(ctx, "presence update error",
					slog.String("presence_id", p.Presence.Id),
					slog.String("error", err.Error()),
				)

				continue
			}

			presenceUpdated++
			p.SynchronizedAt = timestamppb.Now()
		}

		b.Repository.UpdatePresence(p)
	}

	b.Logger.InfoContext(ctx, "device->backend sync done",
		slog.Int("presence.created", presenceCreated),
		slog.Int("presence.updated", presenceUpdated),
	)

	return nil
}

type PageResponse interface {
	GetNextPageToken() string
}

func pageIterator[T proto.Message, Resp PageResponse](
	getter func(pageToken string) (Resp, error),
	extract func(Resp) []T,
) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		pageToken := ""

		for {
			resp, err := getter(pageToken)
			if err != nil {
				var t T
				yield(t, err)

				return
			}

			items := extract(resp)
			for _, item := range items {
				yield(item, nil)
			}

			nextPageToken := resp.GetNextPageToken()
			if nextPageToken == "" {
				return
			}

			pageToken = nextPageToken
		}
	}
}

func collect[T proto.Message](iterator iter.Seq2[T, error]) ([]T, error) {
	var items []T

	for item, err := range iterator {
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}
