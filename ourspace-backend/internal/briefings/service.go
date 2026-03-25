package briefings

import(
	 pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/status"
	"github.com/google/uuid"
	"context"
	"time"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/emptypb"
)
	// 

// 
// 
// "google.golang.org/grpc/codes"
// "google.golang.org/protobuf/proto"
// 

//pb "github.com/cfhn/our-space/ourspace-backend/proto"
	


type Service struct {
	repo *Postgres
}


func NewService(repo *Postgres) * Service {
	return &Service{repo: repo}
}


func (s *Service) UpdateBriefingType(ctx context.Context, request *pb.UpdateBriefingTypeRequest) (*pb.BriefingType, error) {
	fieldViolations, err := s.validateUpdateBriefingType(ctx, request)
	if err != nil {
		return nil, status.Internal(err)
	}
	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	updated, err := s.repo.UpdateBriefingType(ctx, request.BriefingType, request.FieldMask)
	if err != nil{
		return nil, err
	}
	return updated, nil

}


func (s Service) CreateBriefingType(ctx context.Context, request *pb.CreateBriefingTypeRequest) (*pb.BriefingType, error) {
	fieldViolations, err := s.validateCreateBriefingType(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	if request.BriefingTypeId != "" {
		request.BriefingType.Id = request.BriefingTypeId
	} else {
		request.BriefingType.Id = uuid.New().String()
	}

	card, err := s.repo.CreateBriefingType(ctx, request.BriefingType)
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
	
	if request.BriefingType.ExpiresAfter.AsDuration() < 1 * time.Hour || request.BriefingType.ExpiresAfter.AsDuration() > 4 * 365 * 24 * time.Hour{
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field: "briefingType.expires_after",
			Description: "expires_after must be between 1 hour - 4 years",
			Reason: "FIELD_INVALID",
		})
	}
	return fieldViolations, nil
}

func (s *Service) validateUpdateBriefingType(
	ctx context.Context, request *pb.UpdateBriefingTypeRequest,
	) ([]*errdetails.BadRequest_FieldViolation, error) {
	if !request.FieldMask.IsValid(&pb.BriefingType{}) {
		return [] *errdetails.BadRequest_FieldViolation{{
			Field:       "field_mask",
			Description: "invalid_field_mask",
			Reason:      "FIELD_INVALID",
		}}, nil
	}

	fieldViolations := make([]*errdetails.BadRequest_FieldViolation, 0)

	for _, path := range request.FieldMask.Paths {
		switch path {
		case "display_name":
			if len(request.BriefingType.DisplayName) == 0 {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:  "briefing_type.display_name",
					Description: "display_name must have length > 0",
					Reason: "FIELD_INVALID",
					LocalizedMessage: nil,
				})
			}

			if len(request.BriefingType.DisplayName) > 1024 {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:  "briefing_type.display_name",
					Description: "display_name must not be over 1KB",
					Reason: "FIELD_TOO_BIG",
				})
			}
		case "description":
			if len(request.BriefingType.Description) == 0 {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:  "briefing_type.description",
					Description: "description must have length > 0",
					Reason: "FIELD_INVALID",
					LocalizedMessage: nil,
				})
			}
			if len(request.BriefingType.Description) > 10240 {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:  "briefing_type.description",
					Description: "description must have length < 10 KB",
					Reason: "FIELD_INVALID",
					LocalizedMessage: nil,
				})
			}
		}
	// todo add more validations	
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

// func (s* Service) ListBriefingTypes(ctx context.Context, request *pb.UpdateBriefingTypeRequest)(*pb.ListBriefingTypesResponse, error){

// }

