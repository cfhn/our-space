package sync

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/setup"
)

var ErrUnknownLoginOutcome = errors.New("unknown login outcome")

type Repository interface {
	Replace(members []*pbBackend.Member, cards []*pbBackend.Card)
}

type BackendSynchronizer struct {
	AuthClient   pbBackend.AuthServiceClient
	MemberClient pbBackend.MemberServiceClient
	CardClient   pbBackend.CardServiceClient

	Repository Repository
	Logger     *slog.Logger

	ApiKey string
}

func (b *BackendSynchronizer) Synchronize(ctx context.Context) error {
	b.Logger.InfoContext(ctx, "starting sync")

	backendAuth, err := setup.NewBearerTokenAuth(func(ctx context.Context) (string, error) {
		loginResp, err := b.AuthClient.Login(ctx, &pbBackend.LoginRequest{
			Credentials: &pbBackend.LoginRequest_ApiKey{
				ApiKey: &pbBackend.LoginApiKey{
					ApiKey: b.ApiKey,
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
		return b.MemberClient.ListMembers(ctx, &pbBackend.ListMembersRequest{
			PageToken: pageToken,
			PageSize:  100,
		}, grpc.PerRPCCredentials(backendAuth))
	}, (*pbBackend.ListMembersResponse).GetMembers))
	if err != nil {
		return err
	}

	cards, err := collect(pageIterator(func(pageToken string) (*pbBackend.ListCardsResponse, error) {
		return b.CardClient.ListCards(ctx, &pbBackend.ListCardsRequest{
			PageSize:  100,
			PageToken: pageToken,
		}, grpc.PerRPCCredentials(backendAuth))
	}, (*pbBackend.ListCardsResponse).GetCards))
	if err != nil {
		return err
	}

	b.Repository.Replace(members, cards)

	b.Logger.InfoContext(ctx, "sync done", slog.Int("members", len(members)), slog.Int("cards", len(cards)))

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
