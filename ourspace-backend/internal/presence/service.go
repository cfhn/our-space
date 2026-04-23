package presence

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/status"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var ErrFieldUnknown = errors.New("unknown field")

type Service struct {
	repo *Postgres
	pb.UnimplementedPresenceServiceServer
}

func NewService(repo *Postgres) *Service {
	return &Service{repo: repo}
}

func (s Service) Checkin(ctx context.Context, request *pb.CheckinRequest) (*pb.Presence, error) {
	_, fieldViolations := validateCheckinRequest(request)
	if fieldViolations != nil {
		return nil, status.FieldViolations(fieldViolations)
	}

	presence, err := s.repo.CreatePresence(ctx, request.MemberId)
	if err != nil {
		return nil, status.Internal(err)
	}
	return presence, nil
}
func validateCheckinRequest(request *pb.CheckinRequest) (bool, []*errdetails.BadRequest_FieldViolation) {
	return validateMemberId(request.MemberId)
}

func validateMemberId(memberId string) (bool, []*errdetails.BadRequest_FieldViolation) {
	fieldViolations := make([]*errdetails.BadRequest_FieldViolation, 0)
	_, err := uuid.Parse(memberId)
		if memberId == "" {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member_id",
			Description: "member_id field must not be empty",
			Reason:      "FIELD_EMPTY",
		})}else if err != nil {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member_id",
			Description: "member_id must be a valid UUID",
			Reason:      "INVALID_FORMAT",
	})}
	if fieldViolations != nil{
		return false, fieldViolations

	}
			
	return true, nil
}

func (s Service) Checkout(ctx context.Context, request *pb.CheckoutRequest) (*pb.Presence, error) {
	_, fieldViolations := validateCheckoutRequest(request)
	if fieldViolations != nil {
		return nil, status.FieldViolations(fieldViolations)
	}

	presence, err := s.repo.CheckoutPresence(ctx, request.MemberId)
	if err != nil {
		return nil, status.Internal(err)
	}

	return presence, nil
}
func validateCheckoutRequest(request *pb.CheckoutRequest) (bool, []*errdetails.BadRequest_FieldViolation) {
	return validateMemberId(request.MemberId)
}

func (s Service) ListPresences(ctx context.Context, request *pb.ListPresencesRequest) (*pb.ListPresencesResponse, error) {
	pageTokenBytes, err := base64.RawURLEncoding.DecodeString(request.PageToken)
	if err != nil {
		return nil, err
	}

	pageToken := &pb.PresencePageToken{}

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

	presences, err := s.repo.ListPresences(ctx, pageSize+1, pageToken, filters, pageToken.Field)
	if err != nil {
		return nil, err
	}

	var nextPageToken string

	if len(presences) > int(pageSize) {
		presences = presences[:pageSize]

		field := request.SortBy
		if pageToken.Field != pb.PresenceField(pb.PresenceField_PRESENCE_FIELD_UNKNOWN) {
			field = pageToken.Field
		}

		direction := pb.SortDirection_SORT_DIRECTION_ASCENDING
		if pageToken.Direction != pb.SortDirection_SORT_DIRECTION_DEFAULT {
			direction = pageToken.Direction

			lastValue, err := getFieldValue(presences[pageSize-1], field)
			if err != nil {
				return nil, err
			}

			pbNextPageToken := &pb.PresencePageToken{
				Field:     field,
				LastValue: lastValue,
				Direction: direction,
				LastId:    presences[pageSize-1].Id,
			}

			nextPageTokenBytes, err := proto.Marshal(pbNextPageToken)
			if err != nil {
				return nil, err
			}

			nextPageToken = base64.RawURLEncoding.EncodeToString(nextPageTokenBytes)
		}
	}

	return &pb.ListPresencesResponse{
		Presence:      presences,
		NextPageToken: nextPageToken,
	}, nil

}
func getFieldValue(presence *pb.Presence, field pb.PresenceField) (string, error) {
	switch field {
	case pb.PresenceField_PRESENCE_FIELD_ID:
		return presence.Id, nil
	case pb.PresenceField_PRESENCE_FIELD_CHECKIN_TIME:
		return presence.CheckinTime.AsTime().Format(time.RFC3339), nil
	case pb.PresenceField_PRESENCE_FIELD_CHECKOUT_TIME:
		return presence.CheckoutTime.AsTime().Format(time.RFC3339), nil
	case pb.PresenceField_PRESENCE_FIELD_MEMBER_ID:
		return presence.MemberId, nil
	default:
		return "", ErrFieldUnknown
	}
}

func (s Service) UpdatePresence(ctx context.Context, request *pb.UpdatePresenceRequest) (*pb.Presence, error) {
	_, fieldViolations := validateUpdatePresence(request)
	if fieldViolations != nil {
		return nil, status.FieldViolations(fieldViolations)
	}
	presence, err := s.repo.UpdatePresence(ctx, request.GetPresence(), request.FieldMask)
	if err != nil {
		return nil, status.Internal(err)
	}
	return presence, nil
}

func validateUpdatePresence(request *pb.UpdatePresenceRequest) (bool, []*errdetails.BadRequest_FieldViolation) {
	fieldViolations := make([]*errdetails.BadRequest_FieldViolation, 0)
	for _, path := range request.FieldMask.Paths {
		switch path {
		case "member_id":
			_, err := uuid.Parse(request.Presence.MemberId)
			if err != nil {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "presence.member_id",
			Description: "member_id field must not be empty",
			Reason:      "INVALID_MEMBERID",
		})
	}
		case "checkin_time":
	if request.Presence.CheckinTime == nil {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "presence.checkin_time",
			Description: "checkin_time must be set",
			Reason:      "FIELD_EMPTY",
		})
		} else if request.Presence.CheckinTime.AsTime().Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.membership_start",
					Description: "membership_start must after the year 1900",
					Reason:      "FIELD_INVALID",
		})
		} else if request.Presence.CheckinTime.AsTime().After(time.Now().Add(15 * time.Minute)) {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "member.membership_start",
				Description: "membership_start must not be in the future",
				Reason:      "FIELD_INVALID",
			})
	}
		case "checkout_time":
			if request.Presence.CheckinTime == nil {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "presence.checkout_time",
					Description: "checkout_time must be set",
					Reason:      "FIELD_EMPTY",
		})
		} else if request.Presence.CheckoutTime.AsTime().Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.membership_start",
					Description: "membership_start must after the year 1900",
					Reason:      "FIELD_INVALID",
		})
		} else if request.Presence.CheckoutTime.AsTime().After(time.Now().Add(15 * time.Minute)) {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "member.membership_start",
				Description: "membership_start must not be in the future",
				Reason:      "FIELD_INVALID",
			})
	}

		}
	}
	return true, fieldViolations

}

func (s Service) DeletePresence(ctx context.Context, request *pb.DeletePresenceRequest) (*emptypb.Empty, error) {
	err := s.repo.DeletePresence(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
