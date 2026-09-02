package firmware

import (
	"context"
	"encoding/hex"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
	pb "github.com/cfhn/our-space/ourspace-firmware/proto"
	"github.com/cfhn/our-space/pkg/status"
)

type Repository interface {
	FindCardByRFID(rfidValue []byte) *pbBackend.Card
	FindMemberByID(id string) *pbBackend.Member
	FindActivePresence(memberId string) *pb.LocalPresence
	CreatePresence(presence *pb.LocalPresence)
	UpdatePresence(presence *pb.LocalPresence)
}

type Service struct {
	pb.UnimplementedFirmwareServiceServer

	logger *slog.Logger

	notifier *Notifier[pb.ListenForCardEventsResponse]
	repo     Repository
}

func NewService(logger *slog.Logger, repo Repository) *Service {
	svc := &Service{
		logger:   logger,
		notifier: NewNotifier[pb.ListenForCardEventsResponse](),
		repo:     repo,
	}

	return svc
}

func (svc *Service) ScanCard(ctx context.Context, req *pb.ScanCardRequest) (*pb.ScanCardResponse, error) {
	rfidBytes, err := hex.DecodeString(req.CardSerial)
	if err != nil {
		return nil, status.FieldViolations([]*errdetails.BadRequest_FieldViolation{
			{
				Field:       "card_serial",
				Description: "invalid encoding",
			},
		})
	}

	card := svc.repo.FindCardByRFID(rfidBytes)
	if card == nil {
		return &pb.ScanCardResponse{
			Outcome: "card-not-found",
		}, nil
	}

	member := svc.repo.FindMemberByID(card.MemberId)
	if member == nil {
		return &pb.ScanCardResponse{
			Outcome: "member-not-found",
		}, nil
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

	scanCardEvent := &pb.ListenForCardEventsResponse{
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
	}

	svc.notifier.Notify(scanCardEvent)

	return &pb.ScanCardResponse{
		Outcome: "checkin",
	}, nil
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
