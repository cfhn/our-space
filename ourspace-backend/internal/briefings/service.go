package briefings

import  (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/cfhn/out-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/status"
)


type Service struct {
	repo *Postgres
}


func NewService(repo *Postgres) * Service {
	return &Service{repo: repo}
}


func (s Service) CreateBriefingType(ctx context.Context, request *pb.CreateBriefingTypeRequest) (*pb.BriefingType) {
	fieldViolations, err := s.validateCreateBriefingType(ctx, request)
	if err != nil {
		return nil, status.Internal(err)
	}

	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	if request.BriefingTypeId != "" {
		request.BriefingType.Id = request.Id
	} else {
		request.Card.Id = uuid.New().String()
	}

	card, err := s.repo.CreateBriefingType(ctx, request.BriefingType)
	if err != nil{
		return nil, status.Internal(err)
	}
	return card, nil
}

func (s *Service) validateCreateBriefingType (ctx context.Context, request *pb.CreateBriefingTypeRequest,) ([]*errdetails.BadRequest_FieldViolation, error) {
	if request.Briefing == nil {
		return []*errdetails.BadRequest_FieldViolation{
			Field: "briefingType",
			Description: "briefingType must not be empty",
			Reason: "FIELD_EMPTY"
		}
	}, nil

	var fieldViolations []*errdetails.BadRequest_FieldViolation

	if request.display_name == ""{
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.display_name",
			Description: "display_name must be a valid string"
			Reason: "FIELD_EMPTY"
		})
	}
	if len(request.display_name) > 1024 {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.display_name"
			Description: "display_name length must be less than 1024"
			Reason: "FIELD_TOO_BIG"
		})
	}

	if request.description == ""{
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.description",
			Description: "description must be a valid string"
			Reason: "FIELD_EMPTY"
		})
	}
	if len(request.description) > 10240 {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.description"
			Description: "description length must be less than 10240"
			Reason: "FIELD_TOO_BIG"
		})
	}
	
	if request.expires_after < 1 * time.Hour or request.expires_after > 4 * time.year{
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.expires_after"
			Description: "expires_after must be between 1 hour - 4 years"
			Reason: "FIELD_INVALID"
		})
	}
	return fieldViolations, nil
}

func (s *Service) DeleteBriefingType(ctx context.Context, request *pb.DeleteBriefingTypeRequest) (*emptypb.Empty, error){
	err := s.repo.DeleteBriefingType(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	// todo: set to inactive instead if there exist briefings of this type?
	return &emptypb.Empty{}, nil
}



