package members

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
)

var ErrNotFound = errors.New("member not found")

//nolint:gochecknoglobals // constant lookup maps
var (
	membershipFields = map[pb.MemberField]string{
		pb.MemberField_MEMBER_FIELD_ID:               "members.id",
		pb.MemberField_MEMBER_FIELD_NAME:             "members.name",
		pb.MemberField_MEMBER_FIELD_MEMBERSHIP_START: "members.membership_start",
		pb.MemberField_MEMBER_FIELD_MEMBERSHIP_END:   "members.membership_end",
	}
	membershipAttributeFields = map[pb.MemberAttributeField]string{
		pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_ID:             "member_attributes.id",
		pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_TECHNICAL_NAME: "member_attributes.technical_name",
		pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_DISPLAY_NAME:   "member_attributes.display_name",
		pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_TYPE:           "member_attributes.type",
	}
)

type Filters struct {
	NameContains          string
	MembershipStartAfter  time.Time
	MembershipStartBefore time.Time
	MembershipEndAfter    time.Time
	MembershipEndBefore   time.Time
	AgeCategoryEquals     pb.AgeCategory
	TagsContain           []string
}

type Postgres struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) CreateMember(ctx context.Context, member *pb.Member) (*pb.Member, error) {
	var membershipEnd sql.Null[time.Time]

	if member.MembershipEnd != nil {
		membershipEnd = sql.Null[time.Time]{V: member.MembershipEnd.AsTime(), Valid: true}
	}

	tags := []string{}
	if member.Tags != nil {
		tags = member.Tags
	}

	additionalAttributes := map[string]string{}
	if member.AdditionalAttributes != nil {
		additionalAttributes = member.GetAdditionalAttributes()
	}

	additionalAttributesJSON, err := json.Marshal(additionalAttributes)
	if err != nil {
		return nil, err
	}

	_, err = p.db.ExecContext(ctx, `
		insert into members (id, name, membership_start, membership_end, age_category, tags, additional_attributes)
		values ($1, $2, $3, $4, $5, $6, $7::jsonb);
	`, member.Id, member.Name, member.MembershipStart.AsTime(), membershipEnd, member.AgeCategory.String(), tags, additionalAttributesJSON)
	if err != nil {
		return nil, err
	}

	if member.MemberLogin != nil {
		_, err = p.db.ExecContext(ctx, `
			insert into members_auth (id, username, password_hash)
			values ($1, $2, $3)
		`, member.Id, member.MemberLogin.Username, member.MemberLogin.Password)
		if err != nil {
			return nil, err
		}
	}

	return p.GetMember(ctx, member.Id)
}

func (p *Postgres) GetMember(ctx context.Context, id string) (*pb.Member, error) {
	row := p.db.QueryRowContext(ctx, `
		select members.id, name, membership_start, membership_end, age_category, tags, additional_attributes, members_auth.username
		from members
		left join members_auth on members.id = members_auth.id
		where members.id = $1`, id,
	)

	member, err := scanMember(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return member, nil
}

func (p *Postgres) ListMembers(
	ctx context.Context, pageSize int32, token *pb.MemberPageToken, sortField pb.MemberField,
	sortDirection pb.SortDirection, filters *Filters,
) ([]*pb.Member, error) {
	var (
		nameContains          = wrapIlike(filters.NameContains)
		membershipStartBefore = sql.Null[time.Time]{V: filters.MembershipStartBefore, Valid: !filters.MembershipStartBefore.IsZero()}
		membershipStartAfter  = sql.Null[time.Time]{V: filters.MembershipStartAfter, Valid: !filters.MembershipStartAfter.IsZero()}
		membershipEndBefore   = sql.Null[time.Time]{V: filters.MembershipEndBefore, Valid: !filters.MembershipEndBefore.IsZero()}
		membershipEndAfter    = sql.Null[time.Time]{V: filters.MembershipEndAfter, Valid: !filters.MembershipEndAfter.IsZero()}
		ageCategoryEquals     = sql.Null[string]{V: filters.AgeCategoryEquals.String(), Valid: filters.AgeCategoryEquals != pb.AgeCategory_AGE_CATEGORY_UNKNOWN}
	)

	values := append(
		make([]any, 0, 10),
		nameContains,
		membershipStartBefore,
		membershipStartAfter,
		membershipEndBefore,
		membershipEndAfter,
		ageCategoryEquals,
		pgtype.FlatArray[string](filters.TagsContain),
		pageSize,
	)

	paginationCondition, paginationValues := generatePaginationQuery(token, len(values)+1)

	values = append(values, paginationValues...)

	//nolint:gosec // manual concatenation is fine here, uses bound placeholders
	rows, err := p.db.QueryContext(ctx, `
		select members.id, name, membership_start, membership_end, age_category, tags, additional_attributes, members_auth.username
		from members
		left join members_auth on members.id = members_auth.id
		where
		    ($1::text is null OR name ilike $1)
		and ($2::timestamptz is null OR membership_start < $2)
		and ($3::timestamptz is null OR membership_start > $3)
		and ($4::timestamptz is null OR membership_end < $4)
		and ($5::timestamptz is null OR membership_end > $5)
		and ($6::text is null OR age_category = $6)
		and ($7::text[] is null OR cardinality($7::text[]) = 0 OR tags && $7)
		`+paginationCondition+`
		order by `+getSort(sortField, sortDirection, token)+`
		limit $8
	`, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]*pb.Member, 0, pageSize)

	for rows.Next() {
		member, err := scanMember(rows)
		if err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

func generatePaginationQuery(token *pb.MemberPageToken, offset int) (string, []any) {
	fields := make([]string, 0, 2)
	values := make([]any, 0, 2)
	placeholders := make([]string, 0, 2)

	fieldName, ok := membershipFields[token.Field]
	if !ok {
		return "", nil
	}

	fields = append(fields, fieldName)
	values = append(values, token.LastValue)

	if token.Field != pb.MemberField_MEMBER_FIELD_ID {
		fields = append(fields, "members.id")
		values = append(values, token.LastId)
	}

	for i := range len(fields) {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+offset))
	}

	sort := ">"
	if token.Direction == pb.SortDirection_SORT_DIRECTION_DESCENDING {
		sort = "<"
	}

	return "and (" + strings.Join(fields, ",") + ")" +
		sort + "(" + strings.Join(placeholders, ",") + ")", values
}

func getSort(sortField pb.MemberField, direction pb.SortDirection, token *pb.MemberPageToken) string {
	if token.Field != pb.MemberField_MEMBER_FIELD_UNKNOWN {
		sortField = token.Field
		direction = token.Direction
	}

	fieldName, ok := membershipFields[sortField]
	if !ok {
		return "id"
	}

	order := " ASC"
	if direction == pb.SortDirection_SORT_DIRECTION_DESCENDING {
		order = " DESC"
	}

	return fieldName + order + ", id" + order
}

func wrapIlike(filter string) sql.Null[string] {
	if filter == "" {
		return sql.Null[string]{Valid: false}
	}

	return sql.Null[string]{V: "%" + strings.NewReplacer("%", `\\%`, "_", `\\_`).Replace(filter) + "%", Valid: true}
}

func (p *Postgres) UpdateMember(
	ctx context.Context, member *pb.Member, fieldMask *fieldmaskpb.FieldMask,
) (*pb.Member, error) {
	var (
		name                      sql.Null[string]
		membershipStart           sql.Null[time.Time]
		updateMembershipEnd       bool
		membershipEnd             sql.Null[time.Time]
		ageCategory               sql.Null[string]
		updateTags                bool
		tags                      pgtype.FlatArray[string]
		updateMemberLogin         bool
		additionalPropertyUpdates = map[string]*string{}
	)

	for _, path := range fieldMask.Paths {
		switch path {
		case "name":
			name = sql.Null[string]{V: member.Name, Valid: true}
		case "membership_start":
			membershipStart = sql.Null[time.Time]{V: member.MembershipStart.AsTime(), Valid: true}
		case "membership_end":
			updateMembershipEnd = true

			if member.MembershipEnd == nil {
				membershipEnd = sql.Null[time.Time]{}
			} else {
				membershipEnd = sql.Null[time.Time]{V: member.MembershipEnd.AsTime(), Valid: true}
			}
		case "age_category":
			ageCategory = sql.Null[string]{V: member.AgeCategory.String(), Valid: true}
		case "tags":
			updateTags = true

			tags = member.Tags
			if tags == nil {
				tags = pgtype.FlatArray[string]{}
			}
		case "member_login":
			updateMemberLogin = true
		}

		if field, ok := strings.CutPrefix(path, "additional_attributes."); ok {
			value, ok := member.AdditionalAttributes[field]
			if ok {
				additionalPropertyUpdates[field] = &value
			} else {
				additionalPropertyUpdates[field] = nil
			}
		}
	}

	values := append(
		make([]any, 0, 10),
		member.Id,
		name,
		membershipStart,
		updateMembershipEnd,
		membershipEnd,
		ageCategory,
		updateTags,
		tags,
	)

	var (
		addPropSQL    string
		addPropValues []any
	)

	if len(additionalPropertyUpdates) != 0 {
		addPropSQL, addPropValues = generateAdditionalAttributesUpdate(additionalPropertyUpdates, len(values)+1)
	}

	values = append(values, addPropValues...)

	//nolint:gosec // manual concatenation is fine here, uses bound placeholders
	_, err := p.db.ExecContext(ctx, `
		update members
		set
			name = coalesce($2, name),
			membership_start = coalesce($3, membership_start),
			membership_end = case when $4 then $5 else membership_end end,
			age_category = coalesce($6, age_category),
			tags = case when $7 then $8 else tags end,
			`+addPropSQL+`
		where id = $1
	`, values...)
	if err != nil {
		return nil, err
	}

	if updateMemberLogin && member.MemberLogin != nil {
		_, err = p.db.ExecContext(ctx, `
			insert into members_auth (id, username, password_hash)
			values ($1, $2, $3)
			on conflict (id) do update
				set username = excluded.username,
					password_hash = excluded.password_hash
		`, member.Id, member.MemberLogin.Username, member.MemberLogin.Password)
		if err != nil {
			return nil, err
		}
	} else if updateMemberLogin {
		_, err = p.db.ExecContext(ctx, `
			delete from members_auth
			where id = $1
		`, member.Id)
		if err != nil {
			return nil, err
		}
	}

	return p.GetMember(ctx, member.Id)
}

func generateAdditionalAttributesUpdate(updates map[string]*string, offset int) (string, []any) {
	changeFieldTemplate := `jsonb_set(%%s, ARRAY[$%d]::text[], $%d)`
	removeFieldTemplate := `%%s - $%d`

	statements := make([]string, 0, len(updates))
	values := make([]any, 0, len(updates))

	fieldIndex := offset

	for field, value := range updates {
		if value != nil {
			statements = append(statements, fmt.Sprintf(changeFieldTemplate, fieldIndex, fieldIndex+1))
			values = append(values, field, fmt.Sprintf("%q", *value))
			fieldIndex += 2
		} else {
			statements = append(statements, fmt.Sprintf(removeFieldTemplate, fieldIndex))
			values = append(values, field)
			fieldIndex++
		}
	}

	sqlStmt := "%s"

	for _, stmt := range statements {
		sqlStmt = fmt.Sprintf(sqlStmt, stmt)
	}

	sqlStmt = fmt.Sprintf(sqlStmt, "additional_attributes")

	return "additional_attributes = " + sqlStmt, values
}

func (p *Postgres) DeleteMember(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, `delete from members where id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}

func (p *Postgres) ListMemberTags(
	ctx context.Context, pageSize int32, pageToken *pb.MemberTagsPageToken,
) ([]string, error) {
	rows, err := p.db.QueryContext(ctx, `
		select distinct unnest(tags)
		from members
		order by 1
		limit $1
		offset $2;
`, pageSize, pageToken.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]string, 0, pageSize)

	for rows.Next() {
		var tag string

		err = rows.Scan(&tag)
		if err != nil {
			return nil, err
		}

		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

type scanner interface {
	Scan(values ...any) error
}

func scanMember(in scanner) (*pb.Member, error) {
	var (
		member               = &pb.Member{}
		membershipStart      time.Time
		membershipEnd        sql.Null[time.Time]
		ageCategory          string
		username             sql.Null[string]
		additionalProperties string
	)

	m := pgtype.NewMap()

	err := in.Scan(
		&member.Id,
		&member.Name,
		&membershipStart,
		&membershipEnd,
		&ageCategory,
		m.SQLScanner(&member.Tags),
		&additionalProperties,
		&username,
	)
	if err != nil {
		return nil, err
	}

	member.MembershipStart = timestamppb.New(membershipStart)
	member.AgeCategory = pb.AgeCategory(pb.AgeCategory_value[ageCategory])

	if membershipEnd.Valid {
		member.MembershipEnd = timestamppb.New(membershipEnd.V)
	}

	if username.Valid {
		member.MemberLogin = &pb.MemberLogin{
			Username: username.V,
		}
	}

	var additionalPropertiesMap map[string]string

	err = json.Unmarshal([]byte(additionalProperties), &additionalPropertiesMap)
	if err != nil {
		return nil, err
	}

	member.AdditionalAttributes = additionalPropertiesMap

	return member, nil
}

func (p *Postgres) CreateMemberAttribute(
	ctx context.Context, attribute *pb.MemberAttribute,
) (*pb.MemberAttribute, error) {
	_, err := p.db.ExecContext(ctx, `
		insert into member_attributes (id, technical_name, display_name, type, description)
		values ($1, $2, $3, $4, $5);
	`, attribute.GetId(), attribute.GetTechnicalName(), attribute.GetDisplayName(), attribute.GetType().String(), attribute.GetDescription())
	if err != nil {
		return nil, err
	}

	return p.GetMemberAttribute(ctx, attribute.Id)
}

func (p *Postgres) GetMemberAttribute(ctx context.Context, id string) (*pb.MemberAttribute, error) {
	return scanMemberAttribute(p.db.QueryRowContext(
		ctx,
		`
		select id, technical_name, display_name, type, description
		from member_attributes
		where id = $1`,
		id,
	))
}

func (p *Postgres) ListMemberAttributes(
	ctx context.Context, pageSize int32, token *pb.MemberAttributePageToken, sortField pb.MemberAttributeField,
	sortDirection pb.SortDirection,
) ([]*pb.MemberAttribute, error) {
	values := append(
		make([]any, 0, 3),
		pageSize,
	)

	paginationCondition, paginationValues := generateMemberAttributePaginationQuery(token, len(values)+1)

	values = append(values, paginationValues...)

	//nolint:gosec // manual concatenation is fine here, uses bound placeholders
	rows, err := p.db.QueryContext(ctx, `
		select id, technical_name, display_name, type, description
		from member_attributes
		where 1=1 `+paginationCondition+`
		order by `+getMemberAttributeSort(sortField, sortDirection, token)+`
		limit $1
	`, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberAttributes := make([]*pb.MemberAttribute, 0, pageSize)

	for rows.Next() {
		attribute, err := scanMemberAttribute(rows)
		if err != nil {
			return nil, err
		}

		memberAttributes = append(memberAttributes, attribute)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memberAttributes, nil
}

func generateMemberAttributePaginationQuery(token *pb.MemberAttributePageToken, offset int) (string, []any) {
	fields := make([]string, 0, 2)
	values := make([]any, 0, 2)
	placeholders := make([]string, 0, 2)

	fieldName, ok := membershipAttributeFields[token.Field]
	if !ok {
		return "", nil
	}

	fields = append(fields, fieldName)
	values = append(values, token.LastValue)

	if token.Field != pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_ID {
		fields = append(fields, membershipAttributeFields[pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_ID])
		values = append(values, token.LastId)
	}

	for i := range len(fields) {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+offset))
	}

	sort := ">"
	if token.Direction == pb.SortDirection_SORT_DIRECTION_DESCENDING {
		sort = "<"
	}

	sqlCondition := fmt.Sprintf("and (%s) %s (%s)", strings.Join(fields, ","), sort, strings.Join(placeholders, ","))

	return sqlCondition, values
}

func getMemberAttributeSort(
	sortField pb.MemberAttributeField, direction pb.SortDirection, token *pb.MemberAttributePageToken,
) string {
	if token.Field != pb.MemberAttributeField_MEMBER_ATTRIBUTE_FIELD_UNKNOWN {
		sortField = token.Field
		direction = token.Direction
	}

	fieldName, ok := membershipAttributeFields[sortField]
	if !ok {
		return "member_attributes.id"
	}

	order := " ASC"
	if direction == pb.SortDirection_SORT_DIRECTION_DESCENDING {
		order = " DESC"
	}

	return fieldName + order + ", member_attributes.id" + order
}

func scanMemberAttribute(in scanner) (*pb.MemberAttribute, error) {
	var (
		attribute     = &pb.MemberAttribute{}
		attributeType string
	)

	err := in.Scan(
		&attribute.Id,
		&attribute.TechnicalName,
		&attribute.DisplayName,
		&attributeType,
		&attribute.Description,
	)
	if err != nil {
		return nil, err
	}

	attribute.Type = pb.MemberAttribute_Type(pb.MemberAttribute_Type_value[attributeType])

	return attribute, nil
}

func (p *Postgres) UpdateMemberAttribute(
	ctx context.Context, attribute *pb.MemberAttribute, fieldMask *fieldmaskpb.FieldMask,
) (*pb.MemberAttribute, error) {
	var (
		displayName sql.Null[string]
		description sql.Null[string]
	)

	for _, path := range fieldMask.Paths {
		switch path {
		case "display_name":
			displayName = sql.Null[string]{V: attribute.DisplayName, Valid: true}
		case "description":
			description = sql.Null[string]{V: attribute.Description, Valid: attribute.Description != ""}
		}
	}

	_, err := p.db.ExecContext(ctx, `
		update members_attributes
		set
			display_name = coalesce($1, display_name),
			description = coalesce($2, description)
		where id = $3
	`, displayName, description, attribute.Id)
	if err != nil {
		return nil, err
	}

	return p.GetMemberAttribute(ctx, attribute.Id)
}

func (p *Postgres) DeleteMemberAttribute(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, `delete from member_attributes where id = $1`, id)
	return err
}
