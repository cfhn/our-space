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
	_, err := s.ValidateRequest(request.MemberId)
	if err != nil {
		return nil, status.Internal(err)
	}

	presence, err := s.repo.CreatePresence(ctx, request.MemberId)
	if err != nil {
		return nil, status.Internal(err)
	}
	return presence, nil
}

// Checkout
func (s Service) Checkout(ctx context.Context, request pb.CheckoutRequest) (*pb.Presence, error) {
	_, err := s.ValidateRequest(request.MemberId)
	if err != nil {
		return nil, status.Internal(err)
	}

	presence, err := s.repo.CheckoutPresence(ctx, request.MemberId)
	if err != nil {
		return nil, status.Internal(err)
	}

	return presence, nil
}
func (s Service) ValidateRequest(member_id string) (bool, error) {
	if member_id == "" {
		validationError := []*errdetails.BadRequest_FieldViolation{{
			Field:       "member_id",
			Description: "member field must not be empty",
			Reason:      "FIELD_EMPTY",
		}}
		return false, status.FieldViolations(validationError)
	}
	return true, nil
}

// ListPresences
func (s Service) ListPresences(ctx context.Context, request pb.ListPresencesRequest) (*pb.ListPresencesResponse, error) {
	//NOT YET IMPLEMENTED
	return nil, errors.ErrUnsupported
}

func (s Service) UpdatePresence(ctx context.Context, request pb.UpdatePresenceRequest) (*pb.Presence, error) {
	s.ValidateRequest(request.Presence.MemberId)
	s.repo.UpdatePresence(ctx, request.GetPresence(), request.GetFieldMask())
	return nil, errors.ErrUnsupported
}

// DeletePresence
func (s Service) DeletePresences(ctx context.Context, request pb.DeletePresenceRequest) (*pb.Member, error) {
	//NOT YET IMPLEMENTED
	return nil, errors.ErrUnsupported
}
