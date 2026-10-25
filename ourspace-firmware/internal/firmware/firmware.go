package firmware

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/ourspace-firmware/internal/cardreader"
	pb "github.com/cfhn/our-space/ourspace-firmware/proto"
)

type Repository interface {
	FindCardByRFID(rfidValue []byte) *pbBackend.Card
	FindMemberByID(id string) *pbBackend.Member
	FindActivePresence(memberID string) *pb.LocalPresence
	CreatePresence(presence *pb.LocalPresence)
	UpdatePresence(presence *pb.LocalPresence)
}

type Service struct {
	pb.UnimplementedFirmwareServiceServer

	logger *slog.Logger

	notifier *Notifier[pb.ListenForCardEventsResponse]
	repo     Repository
}

func NewService(
	ctx context.Context, logger *slog.Logger, repo Repository, readerEvents <-chan cardreader.CardEvent,
) *Service {
	svc := &Service{
		logger:   logger,
		notifier: NewNotifier[pb.ListenForCardEventsResponse](),
		repo:     repo,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-readerEvents:
				svc.processCardEvent(event)
			}
		}
	}()

	return svc
}

func (svc *Service) ListenForCardEvents(
	req *pb.ListenForCardEventsRequest, resp grpc.ServerStreamingServer[pb.ListenForCardEventsResponse],
) error {
	for {
		select {
		case <-resp.Context().Done():
			return nil
		default:
		}

		cardEvent := svc.notifier.Wait()

		select {
		case <-resp.Context().Done():
			return nil
		default:
		}

		err := resp.Send(cardEvent)
		if err != nil {
			return err
		}
	}
}

func (svc *Service) processCardEvent(event cardreader.CardEvent) {
	switch event.Type {
	case cardreader.CardEventTypeAdd:
		card := svc.repo.FindCardByRFID(event.Uid)
		if card == nil {
			return
		}

		member := svc.repo.FindMemberByID(card.MemberId)
		if member == nil {
			return
		}

		activePresence := svc.repo.FindActivePresence(member.Id)
		if activePresence == nil {
			activePresence = &pb.LocalPresence{
				Presence: &pbBackend.Presence{
					Id:           uuid.NewString(),
					MemberId:     member.Id,
					CheckinTime:  timestamppb.Now(),
					CheckoutTime: nil,
				},
				SynchronizedAt: nil,
				ModifiedAt:     timestamppb.Now(),
			}

			svc.repo.CreatePresence(activePresence)
		} else {
			activePresence.Presence.CheckoutTime = timestamppb.Now()
			activePresence.ModifiedAt = timestamppb.Now()

			svc.repo.UpdatePresence(activePresence)
		}

		response := &pb.ListenForCardEventsResponse{
			Event: &pb.ListenForCardEventsResponse_PlaceCard{
				PlaceCard: &pb.PlaceCardEvent{
					Member: &pb.Member{
						Id:   member.Id,
						Name: member.Name,
					},
					Card: &pb.Card{
						Id:        card.Id,
						ValidFrom: card.ValidFrom,
						ValidTo:   card.ValidTo,
					},
					Presence: activePresence.Presence,
				},
			},
		}

		svc.notifier.Notify(response)
	case cardreader.CardEventTypeRemove:
		response := &pb.ListenForCardEventsResponse{
			Event: &pb.ListenForCardEventsResponse_RemoveCard{
				RemoveCard: &pb.RemoveCardEvent{},
			},
		}

		svc.notifier.Notify(response)
	}
}
