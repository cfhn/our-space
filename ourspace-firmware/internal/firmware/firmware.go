package firmware

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbBackend "github.com/cfhn/our-space/ourspace-backend/proto"
	pb "github.com/cfhn/our-space/ourspace-firmware/proto"
	"github.com/cfhn/our-space/pkg/status"
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

	dataDirectory string
}

func NewService(logger *slog.Logger, repo Repository, dataDirectory string) *Service {
	svc := &Service{
		logger:        logger,
		notifier:      NewNotifier[pb.ListenForCardEventsResponse](),
		repo:          repo,
		dataDirectory: dataDirectory,
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

func (svc *Service) CreateConfiguration(
	ctx context.Context, req *pb.CreateConfigurationRequest,
) (*pb.CreateConfigurationResponse, error) {
	if fieldViolations := validateCreateConfigurationRequest(req); fieldViolations != nil {
		return nil, status.FieldViolations(fieldViolations)
	}

	f, err := os.Create(filepath.Join(svc.dataDirectory, "config.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	configBytes, err := protojson.Marshal(req.Configuration)
	if err != nil {
		return nil, err
	}

	_, err = f.Write(configBytes)
	if err != nil {
		return nil, err
	}

	return &pb.CreateConfigurationResponse{}, nil
}

func validateCreateConfigurationRequest(req *pb.CreateConfigurationRequest) []*errdetails.BadRequest_FieldViolation {
	fieldViolations := make([]*errdetails.BadRequest_FieldViolation, 0)

	if req.Configuration == nil {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "configuration",
			Description: "field must not be nil",
		})

		return fieldViolations
	}

	if _, err := url.ParseRequestURI(req.Configuration.BackendUrl); err != nil {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "backend_url",
			Description: fmt.Sprintf("invalid backend url: %v", err),
		})
	}

	if _, err := uuid.Parse(req.Configuration.ApiKey); err != nil {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "api_key",
			Description: fmt.Sprintf("invalid api key, must be a valid UUID: %v", err),
		})
	}

	return fieldViolations
}
