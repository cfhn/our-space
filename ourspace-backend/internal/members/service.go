package members

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/pwhash"
	"github.com/cfhn/our-space/pkg/status"
)

var ErrFieldUnknown = errors.New("unknown field")

//nolint:gochecknoglobals // constant lookup maps/slices
var (
	validAgeCategories  = []pb.AgeCategory{pb.AgeCategory_AGE_CATEGORY_UNDERAGE, pb.AgeCategory_AGE_CATEGORY_ADULT}
	validAttributeTypes = []pb.MemberAttribute_Type{
		pb.MemberAttribute_TYPE_TEXT_SINGLE_LINE,
		pb.MemberAttribute_TYPE_TEXT_MULI_LINE,
		pb.MemberAttribute_TYPE_NUMBER,
		pb.MemberAttribute_TYPE_DATE,
		pb.MemberAttribute_TYPE_DATETIME,
	}
	validTechnicalName = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_]*[a-z0-9])?$`) // lower_camel_case, doesn't start or end with _
)

type Service struct {
	repo *Postgres
	pb.UnimplementedMemberServiceServer
}

func NewService(repo *Postgres) *Service {
	return &Service{repo: repo}
}

func (s Service) CreateMember(ctx context.Context, request *pb.CreateMemberRequest) (*pb.Member, error) {
	memberAttributes, err := s.listAllMemberAttributes(ctx)
	if err != nil {
		return nil, status.Internal(err)
	}

	validationErrors := validateCreateMember(request, memberAttributes)
	if len(validationErrors) != 0 {
		return nil, status.FieldViolations(validationErrors)
	}

	if request.MemberId != "" {
		request.Member.Id = request.MemberId
	} else {
		request.Member.Id = uuid.New().String()
	}

	if request.Member.MemberLogin != nil {
		hash, err := pwhash.Create(request.Member.MemberLogin.Password)
		if err != nil {
			return nil, err
		}

		request.Member.MemberLogin.Password = hash
	}

	member, err := s.repo.CreateMember(ctx, request.Member)
	if err != nil {
		return nil, status.Internal(err)
	}

	return member, nil
}

func validateCreateMember(
	request *pb.CreateMemberRequest, additionalAttributes map[string]*pb.MemberAttribute,
) []*errdetails.BadRequest_FieldViolation {
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

	const membershipStartLeeway = 15 * time.Minute

	switch {
	case request.Member.MembershipStart == nil:
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member.membership_start",
			Description: "membership_start must be set",
			Reason:      "FIELD_EMPTY",
		})
	case request.Member.MembershipStart.AsTime().Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)):
		fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       "member.membership_start",
			Description: "membership_start must after the year 1900",
			Reason:      "FIELD_INVALID",
		})
	case request.Member.MembershipStart.AsTime().After(time.Now().Add(membershipStartLeeway)):
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

	if request.Member.MemberLogin != nil {
		if request.Member.MemberLogin.Username == "" {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "member.member_login.username",
				Description: "username must not be empty",
			})
		}

		if len(request.Member.MemberLogin.Username) > 64 {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "member.member_login.username",
				Description: "username must not be longer than 64 characters",
			})
		}

		if len(request.Member.MemberLogin.Password) < 8 {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "member.member_login.password",
				Description: "password must not be shorter than 8 characters",
			})
		}

		if !strings.ContainsAny(request.Member.MemberLogin.Password, "0123456789") {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "member.member_login.password",
				Description: "password must contain at least one number",
			})
		}
	}

	for field, value := range request.Member.AdditionalAttributes {
		memberAttribute, ok := additionalAttributes[field]
		if !ok {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "additional_attributes",
				Description: fmt.Sprintf("unknown additional attribute %q", field),
			})

			continue
		}

		if len(value) > 4096 {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       fmt.Sprintf("additional_attributes.%s", field),
				Description: "field can't be longer than 4KB",
			})

			continue
		}

		switch memberAttribute.Type {
		case pb.MemberAttribute_TYPE_UNKNOWN:
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       fmt.Sprintf("additional_attributes.%s", field),
				Description: "unknown field type configured",
			})
		case pb.MemberAttribute_TYPE_TEXT_SINGLE_LINE:
			if strings.ContainsAny(value, "\r\n") {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       fmt.Sprintf("additional_attributes.%s", field),
					Description: "single line field can't contain line breaks",
				})
			}
		case pb.MemberAttribute_TYPE_TEXT_MULI_LINE:
			// no additional validations
		case pb.MemberAttribute_TYPE_NUMBER:
			for _, c := range value {
				if c < '0' || c > '9' {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       fmt.Sprintf("additional_attributes.%s", field),
						Description: "numeric field can't contain non-numeric characters",
					})

					break
				}
			}
		case pb.MemberAttribute_TYPE_DATE:
			_, err := time.Parse(time.DateOnly, value)
			if err != nil {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       fmt.Sprintf("additional_attributes.%s", field),
					Description: fmt.Sprintf("invalid date: %v", err.Error()),
				})
			}
		case pb.MemberAttribute_TYPE_DATETIME:
			_, err := time.Parse(time.RFC3339, value)
			if err != nil {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       fmt.Sprintf("additional_attributes.%s", field),
					Description: fmt.Sprintf("invalid date-time: %v", err.Error()),
				})
			}
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
	memberAttributes, err := s.listAllMemberAttributes(ctx)
	if err != nil {
		return nil, status.Internal(err)
	}

	fieldViolations := validateUpdateMember(request, memberAttributes)
	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	if slices.Contains(request.FieldMask.Paths, "member_login") && request.Member.MemberLogin != nil {
		hash, err := pwhash.Create(request.Member.MemberLogin.Password)
		if err != nil {
			return nil, err
		}

		request.Member.MemberLogin.Password = hash
	}

	updated, err := s.repo.UpdateMember(ctx, request.Member, request.FieldMask)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func validateUpdateMember(
	request *pb.UpdateMemberRequest, additionalAttributes map[string]*pb.MemberAttribute,
) []*errdetails.BadRequest_FieldViolation {
	fieldViolations := make([]*errdetails.BadRequest_FieldViolation, 0)

	for _, path := range request.FieldMask.Paths {
		validPath := false

		switch path {
		case "name":
			validPath = true

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
			validPath = true

			const membershipStartLeeway = 15 * time.Minute

			switch {
			case request.Member.MembershipStart == nil:
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.membership_start",
					Description: "membership_start must be set",
					Reason:      "FIELD_EMPTY",
				})
			case request.Member.MembershipStart.AsTime().Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)):
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.membership_start",
					Description: "membership_start must after the year 1900",
					Reason:      "FIELD_INVALID",
				})
			case request.Member.MembershipStart.AsTime().After(time.Now().Add(membershipStartLeeway)):
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.membership_start",
					Description: "membership_start must not be in the future",
					Reason:      "FIELD_INVALID",
				})
			}
		case "membership_end":
			validPath = true

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
			validPath = true

			if !slices.Contains(validAgeCategories, request.Member.AgeCategory) {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "member.age_category",
					Description: fmt.Sprintf("age_category must be in %v", validAgeCategories),
					Reason:      "FIELD_INVALID",
				})
			}
		case "tags":
			validPath = true

			for i := range request.Member.Tags {
				if request.Member.Tags[i] == "" {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       fmt.Sprintf("member.tags[%d]", i),
						Description: "tag must not be empty",
						Reason:      "FIELD_INVALID",
					})
				}
			}
		case "member_login":
			validPath = true

			if request.Member.MemberLogin != nil {
				if request.Member.MemberLogin.Username == "" {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       "member.member_login.username",
						Description: "username must not be empty",
					})
				}

				if len(request.Member.MemberLogin.Username) > 64 {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       "member.member_login.username",
						Description: "username must not be longer than 64 characters",
					})
				}

				if len(request.Member.MemberLogin.Password) < 8 {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       "member.member_login.password",
						Description: "password must not be shorter than 8 characters",
					})
				}

				if !strings.ContainsAny(request.Member.MemberLogin.Password, "0123456789") {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       "member.member_login.password",
						Description: "password must contain at least one number",
					})
				}
			}
		}

		if field, found := strings.CutPrefix(path, "additional_attributes."); found {
			value := request.Member.AdditionalAttributes[field]

			memberAttribute, ok := additionalAttributes[field]
			if !ok {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       "additional_attributes",
					Description: fmt.Sprintf("unknown additional attribute %q", field),
				})

				continue
			}

			validPath = true

			if len(value) > 4096 {
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       fmt.Sprintf("additional_attributes.%s", field),
					Description: "field can't be longer than 4KB",
				})

				continue
			}

			switch memberAttribute.Type {
			case pb.MemberAttribute_TYPE_UNKNOWN:
				fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       fmt.Sprintf("additional_attributes.%s", field),
					Description: "unknown field type configured",
				})
			case pb.MemberAttribute_TYPE_TEXT_SINGLE_LINE:
				if strings.ContainsAny(value, "\r\n") {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       fmt.Sprintf("additional_attributes.%s", field),
						Description: "single line field can't contain line breaks",
					})
				}
			case pb.MemberAttribute_TYPE_TEXT_MULI_LINE:
				// no additional validations
			case pb.MemberAttribute_TYPE_NUMBER:
				for _, c := range value {
					if c < '0' || c > '9' {
						fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
							Field:       fmt.Sprintf("additional_attributes.%s", field),
							Description: "numeric field can't contain non-numeric characters",
						})

						break
					}
				}
			case pb.MemberAttribute_TYPE_DATE:
				_, err := time.Parse(time.DateOnly, value)
				if err != nil {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       fmt.Sprintf("additional_attributes.%s", field),
						Description: fmt.Sprintf("invalid date: %v", err.Error()),
					})
				}
			case pb.MemberAttribute_TYPE_DATETIME:
				_, err := time.Parse(time.RFC3339, value)
				if err != nil {
					fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
						Field:       fmt.Sprintf("additional_attributes.%s", field),
						Description: fmt.Sprintf("invalid date-time: %v", err.Error()),
					})
				}
			}
		}

		if !validPath {
			fieldViolations = append(fieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       "field_mask",
				Description: fmt.Sprintf("unknown path: %q", path),
			})
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

func (s Service) CreateMemberAttribute(
	ctx context.Context, req *pb.CreateMemberAttributeRequest,
) (*pb.MemberAttribute, error) {
	validationErrors := validateCreateMemberAttribute(req)
	if len(validationErrors) != 0 {
		return nil, status.FieldViolations(validationErrors)
	}

	if req.Attribute.Id == "" {
		req.Attribute.Id = uuid.New().String()
	}

	attribute, err := s.repo.CreateMemberAttribute(ctx, req.Attribute)
	if err != nil {
		return nil, status.Internal(err)
	}

	return attribute, nil
}

func validateCreateMemberAttribute(req *pb.CreateMemberAttributeRequest) []*errdetails.BadRequest_FieldViolation {
	if req.Attribute == nil {
		return []*errdetails.BadRequest_FieldViolation{{
			Field:       "attribute",
			Description: "attribute must be set",
		}}
	}

	var violations []*errdetails.BadRequest_FieldViolation

	if req.Attribute.Id != "" {
		_, err := uuid.Parse(req.Attribute.Id)
		if err != nil {
			violations = append(violations, &errdetails.BadRequest_FieldViolation{
				Field:       "attribute.id",
				Description: "must be valid UUID if set",
			})
		}
	}

	if req.Attribute.DisplayName == "" {
		violations = append(violations, &errdetails.BadRequest_FieldViolation{
			Field:       "attribute.display_name",
			Description: "must be set",
		})
	}

	if len(req.Attribute.DisplayName) > 256 {
		violations = append(violations, &errdetails.BadRequest_FieldViolation{
			Field:       "attribute.display_name",
			Description: "must be shorter than 256 characters, use description for longer texts",
		})
	}

	if req.Attribute.TechnicalName == "" {
		violations = append(violations, &errdetails.BadRequest_FieldViolation{
			Field:       "attribute.technical_name",
			Description: "must be set",
		})
	}

	if !validTechnicalName.MatchString(req.Attribute.TechnicalName) {
		violations = append(violations, &errdetails.BadRequest_FieldViolation{
			Field:       "attribute.technical_name",
			Description: "must be lower_camel_case and not start or end with an underscore",
		})
	}

	if !slices.Contains(validAttributeTypes, req.Attribute.Type) {
		violations = append(violations, &errdetails.BadRequest_FieldViolation{
			Field:       "attribute.type",
			Description: "must be a supported type",
		})
	}

	if len(req.Attribute.Description) > 4096 {
		violations = append(violations, &errdetails.BadRequest_FieldViolation{
			Field:       "attribute.description",
			Description: "must be smaller than 4KB",
		})
	}

	return violations
}

func (s Service) GetMemberAttribute(
	ctx context.Context, req *pb.GetMemberAttributeRequest,
) (*pb.MemberAttribute, error) {
	attribute, err := s.repo.GetMemberAttribute(ctx, req.Id)
	if errors.Is(err, ErrNotFound) {
		return nil, status.NotFound()
	}

	if err != nil {
		return nil, status.Internal(err)
	}

	return attribute, nil
}

func (s Service) ListMemberAttributes(
	ctx context.Context, req *pb.ListMemberAttributesRequest,
) (*pb.ListMemberAttributesResponse, error) {
	pageTokenBytes, err := base64.RawStdEncoding.DecodeString(req.PageToken)
	if err != nil {
		return nil, status.FieldViolations([]*errdetails.BadRequest_FieldViolation{{
			Field:       "page_token",
			Description: "invalid token",
		}})
	}

	pageToken := &pb.MemberAttributePageToken{}

	err = proto.Unmarshal(pageTokenBytes, pageToken)
	if err != nil {
		return nil, status.FieldViolations([]*errdetails.BadRequest_FieldViolation{{
			Field:       "page_token",
			Description: "invalid token",
		}})
	}

	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 50
	}

	attributes, err := s.repo.ListMemberAttributes(ctx, pageSize+1, pageToken, req.SortBy, req.SortDirection)
	if err != nil {
		return nil, err
	}

	var nextPageToken string

	if len(attributes) > int(pageSize) {
		attributes = attributes[:pageSize]

		field := pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_ID
		if pageToken.Field != pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_UNKNOWN {
			field = pageToken.Field
		} else if req.SortBy != pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_UNKNOWN {
			field = req.SortBy
		}

		direction := pb.SortDirection_SORT_DIRECTION_ASCENDING
		if pageToken.Direction != pb.SortDirection_SORT_DIRECTION_DEFAULT {
			direction = pageToken.Direction
		} else if req.SortDirection != pb.SortDirection_SORT_DIRECTION_DEFAULT {
			direction = req.SortDirection
		}

		lastValue, err := getMemberAttributeFieldValue(attributes[pageSize-1], field)
		if err != nil {
			return nil, err
		}

		pbNextPageToken := &pb.MemberAttributePageToken{
			Field:     field,
			LastValue: lastValue,
			Direction: direction,
			LastId:    attributes[pageSize-1].Id,
		}

		nextPageTokenBytes, err := proto.Marshal(pbNextPageToken)
		if err != nil {
			return nil, err
		}

		nextPageToken = base64.RawStdEncoding.EncodeToString(nextPageTokenBytes)
	}

	return &pb.ListMemberAttributesResponse{
		Attributes:    attributes,
		NextPageToken: nextPageToken,
	}, nil
}

func getMemberAttributeFieldValue(attribute *pb.MemberAttribute, field pb.MemberAttributeField) (string, error) {
	switch field {
	case pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_ID:
		return attribute.Id, nil
	case pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_DISPLAY_NAME:
		return attribute.DisplayName, nil
	case pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_TYPE:
		return attribute.Type.String(), nil
	case pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_TECHNICAL_NAME:
		return attribute.TechnicalName, nil
	default:
		return "", ErrFieldUnknown
	}
}

func (s Service) listAllMemberAttributes(ctx context.Context) (map[string]*pb.MemberAttribute, error) {
	var (
		memberAttributes = map[string]*pb.MemberAttribute{}
		token            = &pb.MemberAttributePageToken{}
	)

	for {
		attributes, err := s.repo.ListMemberAttributes(ctx, 1000, token, pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_ID, pb.SortDirection_SORT_DIRECTION_ASCENDING)
		if err != nil {
			return nil, err
		}

		if len(attributes) == 0 {
			break
		}

		for _, attribute := range attributes {
			memberAttributes[attribute.TechnicalName] = attribute
		}

		token.LastValue = attributes[len(attributes)-1].Id
		token.Field = pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_ID
	}

	return memberAttributes, nil
}

func (s Service) UpdateMemberAttribute(
	ctx context.Context, req *pb.UpdateMemberAttributeRequest,
) (*pb.MemberAttribute, error) {
	fieldViolations := validateUpdateMemberAttribute(req)
	if len(fieldViolations) != 0 {
		return nil, status.FieldViolations(fieldViolations)
	}

	return s.repo.UpdateMemberAttribute(ctx, req.Attribute, req.FieldMask)
}

func validateUpdateMemberAttribute(req *pb.UpdateMemberAttributeRequest) []*errdetails.BadRequest_FieldViolation {
	if !req.FieldMask.IsValid(&pb.MemberAttribute{}) {
		return []*errdetails.BadRequest_FieldViolation{{
			Field:       "field_mask",
			Description: "unknown fields in field mask",
			Reason:      "FIELD_INVALID",
		}}
	}

	violations := make([]*errdetails.BadRequest_FieldViolation, 0)

	for _, path := range req.FieldMask.Paths {
		switch path {
		case "display_name":
			if req.Attribute.DisplayName == "" {
				violations = append(violations, &errdetails.BadRequest_FieldViolation{
					Field:       "attribute.display_name",
					Description: "must be set",
				})
			}

			if len(req.Attribute.DisplayName) > 256 {
				violations = append(violations, &errdetails.BadRequest_FieldViolation{
					Field:       "attribute.display_name",
					Description: "must be shorter than 256 characters, use description for longer texts",
				})
			}
		case "description":
			if len(req.Attribute.Description) > 4096 {
				violations = append(violations, &errdetails.BadRequest_FieldViolation{
					Field:       "attribute.description",
					Description: "must be smaller than 4KB",
				})
			}
		default:
			violations = append(violations, &errdetails.BadRequest_FieldViolation{
				Field:       "field_mask",
				Description: "non-updatable field in field mask",
				Reason:      "FIELD_INVALID",
			})
		}
	}

	return violations
}

func (s Service) DeleteMemberAttribute(
	ctx context.Context, req *pb.DeleteMemberAttributeRequest,
) (*emptypb.Empty, error) {
	err := s.repo.DeleteMemberAttribute(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
