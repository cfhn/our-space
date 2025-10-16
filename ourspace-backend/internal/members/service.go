package members

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/status"
)

var ErrFieldUnknown = errors.New("unknown field")

var validAgeCategories = []pb.AgeCategory{pb.AgeCategory_AGE_CATEGORY_UNDERAGE, pb.AgeCategory_AGE_CATEGORY_ADULT}

type Service struct {
	repo *Postgres
	pb.UnimplementedMemberServiceServer
}

func NewService(repo *Postgres) *Service {
	return &Service{repo: repo}
}

func (s Service) CreateMember(ctx context.Context, request *pb.CreateMemberRequest) (*pb.Member, error) {
	validationErrors := validateCreateMember(request)
	if len(validationErrors) != 0 {
		return nil, status.FieldViolations(validationErrors)
	}

	if request.MemberId != "" {
		request.Member.Id = request.MemberId
	} else {
		request.Member.Id = uuid.New().String()
	}

	member, err := s.repo.CreateMember(ctx, request.Member)
	if err != nil {
		return nil, status.Internal(err)
	}

	return member, nil
}

func validateCreateMember(request *pb.CreateMemberRequest) []*errdetails.BadRequest_FieldViolation {
	if request.Member == nil {
		return []*errdetails.BadRequest_FieldViolation{{
			Field:       "member",
			Description: "member field must not be empty",
			Reason:      "FIELD_EMPTY",
		}}
	}

	var fieldViolations []*errdetails.BadRequest_FieldViolation

	if request.Member.Name == "" {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member.name",
			Description: "name must not be empty",
			Reason:      "FIELD_EMPTY",
		})
	}

	if len(request.Member.Name) > 1024 {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member.name",
			Description: "name must not be over 1KB",
			Reason:      "FIELD_TOO_LARGE",
		})
	}

	if request.Member.MembershipStart == nil {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member.membership_start",
			Description: "membership_start must be set",
			Reason:      "FIELD_EMPTY",
		})
	} else if request.Member.MembershipStart.AsTime().Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member.membership_start",
			Description: "membership_start must after the year 1900",
			Reason:      "FIELD_INVALID",
		})
	} else if request.Member.MembershipStart.AsTime().After(time.Now().Add(15 * time.Minute)) {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member.membership_start",
			Description: "membership_start must not be in the future",
			Reason:      "FIELD_INVALID",
		})
	}

	if request.Member.MembershipEnd != nil && request.Member.MembershipStart != nil {
		if request.Member.MembershipEnd.AsTime().Before(request.Member.MembershipStart.AsTime()) {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "member.membership_end",
				Description: "membership_end must be after membership_start",
				Reason:      "FIELD_INVALID",
			})
		}
	}

	if !slices.Contains(validAgeCategories, request.Member.AgeCategory) {
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member.age_category",
			Description: fmt.Sprintf("age_category must be in %v", validAgeCategories),
			Reason:      "FIELD_INVALID",
		})
	}

	for i := range request.Member.Tags {
		if request.Member.Tags[i] == "" {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       fmt.Sprintf("member.tags[%d]", i),
				Description: "tag must not be empty",
				Reason:      "FIELD_INVALID",
			})
		}
	}

	if len(fieldViolations) != 0 {
		return fieldViolations
	}

	return nil
}

func (s Service) GetMember(ctx context.Context, request *pb.GetMemberRequest) (*pb.Member, error) {
	member, err := s.repo.GetMember(ctx, request.Id)
	if errors.Is(err, ErrNotFound) {
		return nil, status.NotFound()
	}
	if err != nil {
		return nil, status.Internal(err)
	}

	return member, nil
}

func (s Service) ListMembers(ctx context.Context, request *pb.ListMembersRequest) (*pb.ListMembersResponse, error) {
	pageTokenBytes, err := base64.RawStdEncoding.DecodeString(request.PageToken)
	if err != nil {
		return nil, err
	}

	pageToken := &pb.MemberPageToken{}

	err = proto.Unmarshal(pageTokenBytes, pageToken)
	if err != nil {
		return nil, err
	}

	filters := &Filters{}
	if request.NameContains != nil {
		filters.NameContains = *request.NameContains
	}
	if request.MembershipStartAfter != nil {
		filters.MembershipStartAfter = request.MembershipStartAfter.AsTime()
	}
	if request.MembershipStartBefore != nil {
		filters.MembershipStartBefore = request.MembershipStartBefore.AsTime()
	}
	if request.MembershipEndAfter != nil {
		filters.MembershipEndAfter = request.MembershipEndAfter.AsTime()
	}
	if request.MembershipEndBefore != nil {
		filters.MembershipEndBefore = request.MembershipEndBefore.AsTime()
	}
	if request.AgeCategoryEquals != nil {
		filters.AgeCategoryEquals = *request.AgeCategoryEquals
	}
	if len(request.TagContains) != 0 {
		filters.TagsContain = request.TagContains
	}

	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = 50
	}

	members, err := s.repo.ListMembers(ctx, pageSize+1, pageToken, request.SortBy, request.SortDirection, filters)
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
		} else if request.SortDirection != pb.SortDirection_SORT_DIRECTION_DEFAULT {
			direction = request.SortDirection
		}

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

	return &pb.ListMembersResponse{
		Members:       members,
		NextPageToken: nextPageToken,
	}, nil
}

func (s Service) UpdateMember(ctx context.Context, request *pb.UpdateMemberRequest) (*pb.Member, error) {
	fieldViolations := validateUpdateMember(request)
	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	updated, err := s.repo.UpdateMember(ctx, request.Member, request.FieldMask)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func validateUpdateMember(request *pb.UpdateMemberRequest) []*errdetails.BadRequest_FieldViolation {
	if !request.FieldMask.IsValid(&pb.Member{}) {
		return []*errdetails.BadRequest_FieldViolation{{
			Field:       "field_mask",
			Description: "invalid field_mask",
			Reason:      "FIELD_INVALID",
		}}
	}

	fieldViolations := make([]*errdetails.BadRequest_FieldViolation, 0)

	for _, path := range request.FieldMask.Paths {
		switch path {
		case "name":
			if request.Member.Name == "" {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.name",
					Description: "name must not be empty",
					Reason:      "FIELD_EMPTY",
				})
			}

			if len(request.Member.Name) > 1024 {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.name",
					Description: "name must not be over 1KB",
					Reason:      "FIELD_TOO_LARGE",
				})
			}
		case "membership_start":
			if request.Member.MembershipStart == nil {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.membership_start",
					Description: "membership_start must be set",
					Reason:      "FIELD_EMPTY",
				})
			} else if request.Member.MembershipStart.AsTime().Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.membership_start",
					Description: "membership_start must after the year 1900",
					Reason:      "FIELD_INVALID",
				})
			} else if request.Member.MembershipStart.AsTime().After(time.Now().Add(15 * time.Minute)) {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.membership_start",
					Description: "membership_start must not be in the future",
					Reason:      "FIELD_INVALID",
				})
			}
		case "membership_end":
			if request.Member.MembershipEnd != nil && request.Member.MembershipStart != nil {
				if request.Member.MembershipEnd.AsTime().Before(request.Member.MembershipStart.AsTime()) {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       "member.membership_end",
						Description: "membership_end must be after membership_start",
						Reason:      "FIELD_INVALID",
					})
				}
			}
		case "age_category":
			if !slices.Contains(validAgeCategories, request.Member.AgeCategory) {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.age_category",
					Description: fmt.Sprintf("age_category must be in %v", validAgeCategories),
					Reason:      "FIELD_INVALID",
				})
			}
		case "tags":
			for i := range request.Member.Tags {
				if request.Member.Tags[i] == "" {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       fmt.Sprintf("member.tags[%d]", i),
						Description: "tag must not be empty",
						Reason:      "FIELD_INVALID",
					})
				}
			}
		}
	}

	return fieldViolations
}

func (s Service) DeleteMember(ctx context.Context, request *pb.DeleteMemberRequest) (*emptypb.Empty, error) {
	err := s.repo.DeleteMember(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s Service) ListMemberTags(
	ctx context.Context, request *pb.ListMemberTagsRequest,
) (*pb.ListMemberTagsResponse, error) {
	pageTokenBytes, err := base64.RawStdEncoding.DecodeString(request.PageToken)
	if err != nil {
		return nil, err
	}

	pageSize := request.PageSize
	if pageSize == 0 {
		pageSize = 50
	}

	pageToken := &pb.MemberTagsPageToken{}

	err = proto.Unmarshal(pageTokenBytes, pageToken)
	if err != nil {
		return nil, err
	}

	tags, err := s.repo.ListMemberTags(ctx, request.PageSize+1, pageToken)
	if err != nil {
		return nil, err
	}

	var nextPageToken string

	if len(tags) > int(request.PageSize) {
		tags = tags[:request.PageSize]
		nextPageTokenPb := &pb.MemberTagsPageToken{
			Offset: pageToken.Offset + pageSize,
		}
		nextPageTokenBytes, err := proto.Marshal(nextPageTokenPb)
		if err != nil {
			return nil, err
		}
		nextPageToken = base64.RawStdEncoding.EncodeToString(nextPageTokenBytes)
	}

	return &pb.ListMemberTagsResponse{
		Tags:          tags,
		NextPageToken: nextPageToken,
	}, nil
}

func getFieldValue(member *pb.Member, field pb.MemberField) (string, error) {
	switch field {
	case pb.MemberField_MEMBER_FIELD_ID:
		return member.Id, nil
	case pb.MemberField_MEMBER_FIELD_NAME:
		return member.Name, nil
	case pb.MemberField_MEMBER_FIELD_MEMBERSHIP_START:
		return member.MembershipStart.AsTime().Format(time.RFC3339), nil
	case pb.MemberField_MEMBER_FIELD_MEMBERSHIP_END:
		return member.MembershipEnd.AsTime().Format(time.RFC3339), nil
	default:
		return "", ErrFieldUnknown
	}
}
