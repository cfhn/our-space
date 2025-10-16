package firmware

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
	pb "github.com/cfhn/our-space/ourspace-firmware/proto"
)

type Repository interface {
	FindCardByRFID(rfidValue []byte) *pbBackend.Card
	FindMemberByID(id string) *pbBackend.Member
}

type Service struct {
	pb.UnimplementedFirmwareServiceServer

	logger *slog.Logger

	notifier *Notifier
	repo     Repository
}

func NewService(logger *slog.Logger, repo Repository) *Service {
	svc := &Service{
		logger:   logger,
		notifier: NewNotifier(),
		repo:     repo,
	}

	return svc
}

func (svc *Service) ScanCard(ctx context.Context, req *pb.ScanCardRequest) (*pb.ScanCardResponse, error) {
	svc.notifier.Notify(req.RfidValue)

	return &pb.ScanCardResponse{}, nil
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

		rfidValue := svc.notifier.Wait()

		select {
		case <-resp.Context().Done():
			return nil
		default:
		}

		card := svc.repo.FindCardByRFID(rfidValue)
		if card == nil {
			continue
		}

		member := svc.repo.FindMemberByID(card.MemberId)
		if member == nil {
			continue
		}

		err := resp.Send(&pb.ListenForCardEventsResponse{
			Member: &pb.Member{
				Id:   member.Id,
				Name: member.Name,
			},
			Card: &pb.Card{
				Id:        card.Id,
				ValidFrom: card.ValidFrom,
				ValidTo:   card.ValidTo,
			},
		})

		if err != nil {
			return err
		}
	}
}
