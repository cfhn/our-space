package sync

import (
	"context"
	"iter"
	"log/slog"

	"google.golang.org/protobuf/proto"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
)

type Repository interface {
	Replace(members []*pbBackend.Member, cards []*pbBackend.Card)
}

type BackendSynchronizer struct {
	MemberClient pbBackend.MemberServiceClient
	CardClient   pbBackend.CardServiceClient

	Repository Repository
	Logger     *slog.Logger
}

func (b *BackendSynchronizer) Synchronize(ctx context.Context) error {
	b.Logger.InfoContext(ctx, "starting sync")

	members, err := collect(pageIterator(func(pageToken string) (*pbBackend.ListMembersResponse, error) {
		return b.MemberClient.ListMembers(ctx, &pbBackend.ListMembersRequest{
			PageToken: pageToken,
			PageSize:  100,
		})
	}, (*pbBackend.ListMembersResponse).GetMembers))
	if err != nil {
		return err
	}

	cards, err := collect(pageIterator(func(pageToken string) (*pbBackend.ListCardsResponse, error) {
		return b.CardClient.ListCards(ctx, &pbBackend.ListCardsRequest{
			PageSize:  100,
			PageToken: pageToken,
		})

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
