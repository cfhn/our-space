package briefings

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

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/status"
)


type Service struct {
	repo *Postgres
}


func NewService(repo *Postgres) * Service {
	return &Service{repo: repo}
}

func (s *Service) UpdateBriefingType(ctx context.Context, request *pb.UpdateBriefingTypeRequest) (*pb.BriefingType, error) {
	fieldViolations, err := s.ValidateUpdateBriefingType(ctx, request)
	if err != nil {
		return nil, status.Internal(err)
	}
	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	updated, err := s.repo.UpdateBriefingType(ctx, request.BriefingType)
}


func (s Service) CreateBriefingType(ctx context.Context, request *pb.CreateBriefingTypeRequest) (*pb.BriefingType) {
	fieldViolations, err := s.validateCreateBriefingType(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	if request.BriefingTypeId != "" {
		request.BriefingType.Id = request.Id
	} else {
		request.Card.Id = uuid.New().String()
	}

	card, err := s.repo.CreateBriefingType(ctx, request.BriefingType, request.FieldMask)
	if err != nil{
		return nil, status.Internal(err)
	}
	return card, nil
}

func (s *Service) validateCreateBriefingType (
	ctx context.Context, request *pb.CreateBriefingTypeRequest,
	) ([]*errdetails.BadRequest_FieldViolation, error) {
	if request.BriefingType == nil {
		return []*errdetails.BadRequest_FieldViolation{{
			Field: "briefingType",
			Description: "briefingType must not be empty",
			Reason: "FIELD_EMPTY",
		}}, nil
	}

	var fieldViolations []*errdetails.BadRequest_FieldViolation

	if request.BriefingType.DisplayName == ""{
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.display_name",
			Description: "display_name must be a valid string",
			Reason: "FIELD_EMPTY",
		})
	}
	if len(request.BriefingType.DisplayName) > 1024 {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.display_name",
			Description: "display_name length must be less than 1024",
			Reason: "FIELD_TOO_BIG",
		})
	}

	if request.BriefingType.Description == ""{
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.description",
			Description: "description must be a valid string",
			Reason: "FIELD_EMPTY",
		})
	}
	if len(request.BriefingType.Description) > 10240 {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.description",
			Description: "description length must be less than 10240",
			Reason: "FIELD_TOO_BIG",
		})
	}
	
	if request.BriefingType.ExpiresAfter.AsDuration() < 1 * time.Hour || request.BriefingType.ExpiresAfter.AsDuration() > 4 * time.year{
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.expires_after",
			Description: "expires_after must be between 1 hour - 4 years",
			Reason: "FIELD_INVALID",
		})
	}
	return fieldViolations, nil
}

func (s *Service) ValidateUpdateBriefingType()


func (s *Service) DeleteBriefingType(ctx context.Context, request *pb.DeleteBriefingTypeRequest) (*emptypb.Empty, error){
	err := s.repo.DeleteBriefingType(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	// todo: set to inactive instead if there exist briefings of this type?
	return &emptypb.Empty{}, nil
}



