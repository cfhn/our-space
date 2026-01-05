package presence

import (
	"context"
	"errors"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

var ErrFieldUnknown = errors.New("unknown field")

type Service struct {
	repo *Postgres
	pb.UnimplementedPresenceServiceServer
}

func NewService(repo *Postgres) *Service {
	return &Service{repo: repo}
}

func (s Service) Checkin(ctx context.Context, request pb.CheckinRequest) (*pb.Presence, error) {
	if request.MemberId == "" {
		validationError := []*errdetails.BadRequest_FieldViolation{{
			Field:       "member_id",
			Description: "member field must not be empty",
			Reason:      "FIELD_EMPTY",
		}}
		return nil, status.FieldViolations(validationError)
	}

	presence, err := s.repo.CreatePresence(ctx, request.MemberId)
	if err != nil {
		return nil, status.Internal(err)
	}
	return presence, nil
}

// Checkout
func (s Service) Checkout(ctx context.Context, request pb.CheckoutRequest) (*pb.Presence, error) {
	//Request Validity; Search Presence with Member
	//Get Time
	//Set Checkout Time
	return nil, errors.ErrUnsupported
}

// ListPresences
func (s Service) ListPresences(ctx context.Context, request pb.ListPresencesRequest) (*pb.ListPresencesResponse, error) {
	//NOT YET IMPLEMENTED
	return nil, errors.ErrUnsupported
}

func (s Service) UpdatePresence(ctx context.Context, request pb.UpdatePresenceRequest) (*pb.Presence, error) {
	//NOT YET IMPLEMENTED
	return nil, errors.ErrUnsupported
}

// DeletePresence
func (s Service) DeletePresences(ctx context.Context, request pb.DeletePresenceRequest) (*pb.Member, error) {
	//NOT YET IMPLEMENTED
	return nil, errors.ErrUnsupported
}
