package presence

import (
	"context"
	"encoding/base64"
	"errors"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
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
	pageTokenBytes, err := base64.RawStdEncoding.DecodeString(request.PageToken)
	if err != nil {
		return nil, err
	}

	pageToken := &pb.PageToken

	err = proto.Unmarshal(pageTokenBytes, pageToken)
	if err != nil {
		return nil, err
	}

	filters := &Filters{}
	if request.CheckinTimeBefore != nil {
		filters.CheckinTimeBefore = request.CheckinTimeBefore.AsTime()
	}
	if request.CheckinTimeAfter != nil {
		filters.CheckinTimeAfter = request.CheckinTimeAfter.AsTime()
	}
	if request.CheckoutTimeAfter != nil {
		filters.CheckoutTimeAfter = request.CheckoutTimeAfter.AsTime()
	}
	if request.CheckoutTimeBefore != nil {
		filters.CheckoutTimeBefore = request.CheckoutTimeBefore.AsTime()
	}
	if request.MemberId != nil {
		filters.MemberId = *request.MemberId
	}

	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = 50
	}

	members, err := s.repo.ListMembers(ctx, pageSize+1, pageToken, pageToken.Direction, filters)
	if err != nil {
		return nil, err
	}

	var nextPageToken string

	if len(members) > int(pageSize) {
		members = members[:pageSize]

		field := pb.MemberField_MEMBER_FIELD_ID
		if pageToken.Field != pb.MemberField_MEMBER_FIELD_UNKNOWN {
			field = pageToken.Field
		} else if request.SortBy != pb.MemberField_MEMBER_FIELD_UNKNOWN {
			field = request.SortBy
		}

		direction := pb.SortDirection_SORT_DIRECTION_ASCENDING
		if pageToken.Direction != pb.SortDirection_SORT_DIRECTION_DEFAULT {
			direction = pageToken.Direction

			lastValue, err := getFieldValue(members[pageSize-1], field)
			if err != nil {
				return nil, err
			}

			pbNextPageToken := &pb.MemberPageToken{
				Field:     field,
				LastValue: lastValue,
				Direction: direction,
				LastId:    members[pageSize-1].Id,
			}

			nextPageTokenBytes, err := proto.Marshal(pbNextPageToken)
			if err != nil {
				return nil, err
			}

			nextPageToken = base64.RawStdEncoding.EncodeToString(nextPageTokenBytes)
		}

		return &pb.ListPresencesResponse{
			Presence:      pre,
			NextPageToken: nextPageToken,
		}, nil
	}
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
