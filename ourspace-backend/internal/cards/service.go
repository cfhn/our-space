package cards

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/status"
)

var ErrFieldUnknown = errors.New("unknown field")

type MemberService interface {
	GetMember(ctx context.Context, request *pb.GetMemberRequest) (*pb.Member, error)
}

type Service struct {
	repo          *Postgres
	memberService MemberService
	pb.UnimplementedCardServiceServer
}

func NewService(repo *Postgres, memberService MemberService) *Service {
	return &Service{repo: repo, memberService: memberService}
}

func (s *Service) CreateCard(ctx context.Context, request *pb.CreateCardRequest) (*pb.Card, error) {
	fieldViolations, err := s.validateCreateCard(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	if request.CardId != "" {
		request.Card.Id = request.CardId
	} else {
		request.Card.Id = uuid.New().String()
	}

	card, err := s.repo.CreateCard(ctx, request.Card)
	if err != nil {
		return nil, status.Internal(err)
	}

	return card, nil
}

func (s *Service) validateCreateCard(
	ctx context.Context, request *pb.CreateCardRequest,
) ([]*errdetails.BadRequest_FieldViolation, error) {
	if request.Card == nil {
		return []*errdetails.BadRequest_FieldViolation{{
			Field:       "card",
			Description: "card field must not be empty",
			Reason:      "FIELD_EMPTY",
		}}, nil
	}

	var fieldViolations []*errdetails.BadRequest_FieldViolation

	if request.Card.MemberId == "" {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "card.member_id",
			Description: "card must be assigned to a member",
			Reason:      "FIELD_EMPTY",
		})
	}

	if _, err := uuid.Parse(request.Card.MemberId); err != nil {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:            "card.member_id",
			Description:      "member_id must be a valid UUID",
			Reason:           "FIELD_INVALID",
			LocalizedMessage: nil,
		})
	}

	if request.Card.MemberId != "" {
		_, err := s.memberService.GetMember(ctx, &pb.GetMemberRequest{Id: request.Card.MemberId})
		grpcStatus := status.FromError(err)

		if grpcStatus.Code() == codes.NotFound {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "card.member_id",
				Description: "member does not exist",
				Reason:      "FIELD_INVALID",
			})
		} else if grpcStatus.Code() != codes.OK {
			return nil, err
		}
	}

	if len(request.Card.RfidValue) == 0 {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "card.rfid_value",
			Description: "rfid_value must not be empty",
			Reason:      "FIELD_EMPTY",
		})
	}

	if len(request.Card.RfidValue) > 1024 {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "card.rfid_value",
			Description: "rfid_value must not be over 1KB",
			Reason:      "FIELD_TOO_BIG",
		})
	}

	if request.Card.ValidFrom == nil {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "card.valid_from",
			Description: "valid_from must not be empty",
			Reason:      "FIELD_EMPTY",
		})
	}

	if request.Card.ValidTo == nil {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "card.valid_to",
			Description: "valid_to must not be empty",
			Reason:      "FIELD_EMPTY",
		})
	}

	if request.Card.ValidTo.AsTime().Before(request.Card.ValidFrom.AsTime()) {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "card.valid_to",
			Description: "valid_to must not be before valid_from",
			Reason:      "FIELD_INVALID",
		})
	}

	return fieldViolations, nil
}

func (s *Service) GetCard(ctx context.Context, request *pb.GetCardRequest) (*pb.Card, error) {
	card, err := s.repo.GetCard(ctx, request.Id)
	if errors.Is(err, ErrNotFound) {
		return nil, status.NotFound()
	}
	if err != nil {
		return nil, status.Internal(err)
	}
	return card, nil
}

func (s *Service) ListCards(ctx context.Context, request *pb.ListCardsRequest) (*pb.ListCardsResponse, error) {
	pageTokenBytes, err := base64.RawStdEncoding.DecodeString(request.PageToken)
	if err != nil {
		return nil, err
	}

	pageToken := &pb.CardPageToken{}

	err = proto.Unmarshal(pageTokenBytes, pageToken)
	if err != nil {
		return nil, err
	}

	filters := &Filters{}
	if request.MemberId != "" {
		filters.MemberID = request.MemberId
	}
	if request.ValidOn != nil {
		filters.ValidOn = request.ValidOn.AsTime()
	}
	if len(request.RfidValue) != 0 {
		filters.RfidValue = request.RfidValue
	}

	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = 50
	}

	cards, err := s.repo.ListCards(ctx, pageSize+1, pageToken, request.SortBy, request.SortDirection, filters)
	if err != nil {
		return nil, err
	}

	var nextPageToken string

	if len(cards) > int(pageSize) {
		cards = cards[:pageSize]

		field := pb.CardField_CARD_FIELD_ID
		if pageToken.Field != pb.CardField_CARD_FIELD_UNKNOWN {
			field = pageToken.Field
		} else if request.SortBy != pb.CardField_CARD_FIELD_UNKNOWN {
			field = request.SortBy
		}

		direction := pb.SortDirection_SORT_DIRECTION_ASCENDING
		if pageToken.Direction != pb.SortDirection_SORT_DIRECTION_DEFAULT {
			direction = pageToken.Direction
		} else if request.SortDirection != pb.SortDirection_SORT_DIRECTION_DEFAULT {
			direction = request.SortDirection
		}

		lastValue, err := getFieldValue(cards[pageSize-1], field)
		if err != nil {
			return nil, err
		}

		pbNextPageToken := &pb.CardPageToken{
			Field:     field,
			LastValue: lastValue,
			Direction: direction,
			LastId:    cards[pageSize-1].Id,
		}

		nextPageTokenBytes, err := proto.Marshal(pbNextPageToken)
		if err != nil {
			return nil, err
		}

		nextPageToken = base64.RawStdEncoding.EncodeToString(nextPageTokenBytes)
	}

	return &pb.ListCardsResponse{
		Cards:         cards,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *Service) UpdateCard(ctx context.Context, request *pb.UpdateCardRequest) (*pb.Card, error) {
	fieldViolations, err := s.validateUpdateCard(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	updated, err := s.repo.UpdateCard(ctx, request.Card, request.FieldMask)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) validateUpdateCard(
	ctx context.Context, request *pb.UpdateCardRequest,
) ([]*errdetails.BadRequest_FieldViolation, error) {
	if !request.FieldMask.IsValid(&pb.Card{}) {
		return []*errdetails.BadRequest_FieldViolation{{
			Field:       "field_mask",
			Description: "invalid field_mask",
			Reason:      "FIELD_INVALID",
		}}, nil
	}

	fieldViolations := make([]*errdetails.BadRequest_FieldViolation, 0)

	for _, path := range request.FieldMask.Paths {
		switch path {
		case "member_id":
			if request.Card.MemberId == "" {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "card.member_id",
					Description: "card must be assigned to a member",
					Reason:      "FIELD_EMPTY",
				})
			}

			if _, err := uuid.Parse(request.Card.MemberId); err != nil {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:            "card.member_id",
					Description:      "member_id must be a valid UUID",
					Reason:           "FIELD_INVALID",
					LocalizedMessage: nil,
				})
			}

			if request.Card.MemberId != "" {
				_, err := s.memberService.GetMember(ctx, &pb.GetMemberRequest{Id: request.Card.MemberId})
				grpcStatus := status.FromError(err)

				if grpcStatus.Code() == codes.NotFound {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       "card.member_id",
						Description: "member does not exist",
						Reason:      "FIELD_INVALID",
					})
				} else if grpcStatus.Code() != codes.OK {
					return nil, err
				}
			}
		case "rfid_value":
			if len(request.Card.RfidValue) == 0 {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "card.rfid_value",
					Description: "rfid_value must not be empty",
					Reason:      "FIELD_EMPTY",
				})
			}

			if len(request.Card.RfidValue) > 1024 {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "card.rfid_value",
					Description: "rfid_value must not be over 1KB",
					Reason:      "FIELD_TOO_BIG",
				})
			}
		case "valid_from":
			if request.Card.ValidFrom == nil {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "card.valid_from",
					Description: "valid_from must not be empty",
					Reason:      "FIELD_EMPTY",
				})
			}
		case "valid_to":
			if request.Card.ValidTo == nil {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "card.valid_to",
					Description: "valid_to must not be empty",
					Reason:      "FIELD_EMPTY",
				})
			}
		}
	}

	return fieldViolations, nil
}

func (s *Service) DeleteCard(ctx context.Context, request *pb.DeleteCardRequest) (*emptypb.Empty, error) {
	err := s.repo.DeleteCard(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func getFieldValue(card *pb.Card, field pb.CardField) (string, error) {
	switch field {
	case pb.CardField_CARD_FIELD_ID:
		return card.Id, nil
	case pb.CardField_CARD_FIELD_MEMBER_ID:
		return card.MemberId, nil
	case pb.CardField_CARD_FIELD_VALID_FROM:
		return card.ValidFrom.AsTime().Format(time.RFC3339), nil
	case pb.CardField_CARD_FIELD_VALID_TO:
		return card.ValidTo.AsTime().Format(time.RFC3339), nil
	default:
		return "", ErrFieldUnknown
	}
}
