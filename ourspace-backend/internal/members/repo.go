package members

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/cfhn/our-space/ourspace-backend/pb"
)

var ErrNotFound = errors.New("member not found")

var membershipFields = map[pb.MemberField]string{
	pb.MemberField_MEMBER_FIELD_ID:               "id",
	pb.MemberField_MEMBER_FIELD_NAME:             "name",
	pb.MemberField_MEMBER_FIELD_MEMBERSHIP_START: "membership_start",
	pb.MemberField_MEMBER_FIELD_MEMBERSHIP_END:   "membership_end",
}

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
	var (
		membershipEnd sql.Null[time.Time]
	)

	if member.MembershipEnd != nil {
		membershipEnd = sql.Null[time.Time]{V: member.MembershipEnd.AsTime(), Valid: true}
	}

	_, err := p.db.ExecContext(ctx, `
		insert into members (id, name, membership_start, membership_end, age_category, tags)
		values ($1, $2, $3, $4, $5, $6);
	`, member.Id, member.Name, member.MembershipStart.AsTime(), membershipEnd, member.AgeCategory.String(), member.Tags)

	if err != nil {
		return nil, err
	}

	return p.GetMember(ctx, member.Id)
}

func (p *Postgres) GetMember(ctx context.Context, id string) (*pb.Member, error) {
	row := p.db.QueryRowContext(ctx, `
		select id, name, membership_start, membership_end, age_category, tags
		from members
		where id = $1`, id,
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
	paginationCondition, paginationValues := generatePaginationQuery(token, 9)

	var (
		nameContains          = wrapIlike(filters.NameContains)
		membershipStartBefore = sql.Null[time.Time]{V: filters.MembershipStartBefore, Valid: !filters.MembershipStartBefore.IsZero()}
		membershipStartAfter  = sql.Null[time.Time]{V: filters.MembershipStartAfter, Valid: !filters.MembershipStartAfter.IsZero()}
		membershipEndBefore   = sql.Null[time.Time]{V: filters.MembershipEndBefore, Valid: !filters.MembershipEndBefore.IsZero()}
		membershipEndAfter    = sql.Null[time.Time]{V: filters.MembershipEndAfter, Valid: !filters.MembershipEndAfter.IsZero()}
		ageCategoryEquals     = sql.Null[string]{V: filters.AgeCategoryEquals.String(), Valid: filters.AgeCategoryEquals != pb.AgeCategory_AGE_CATEGORY_UNKNOWN}
	)

	values := []any{
		nameContains,
		membershipStartBefore,
		membershipStartAfter,
		membershipEndBefore,
		membershipEndAfter,
		ageCategoryEquals,
		pgtype.FlatArray[string](filters.TagsContain),
		pageSize,
	}

	values = append(values, paginationValues...)

	rows, err := p.db.QueryContext(ctx, `
		select id, name, membership_start, membership_end, age_category, tags
		from members
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
		fields = append(fields, "id")
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
		name                sql.Null[string]
		membershipStart     sql.Null[time.Time]
		updateMembershipEnd bool
		membershipEnd       sql.Null[time.Time]
		ageCategory         sql.Null[string]
		updateTags          bool
		tags                pgtype.FlatArray[string]
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
		}
	}

	_, err := p.db.ExecContext(ctx, `
		update members
		set
			name = coalesce($2, name),
			membership_start = coalesce($3, membership_start),
			membership_end = case when $4 then $5 else membership_end end,
			age_category = coalesce($6, age_category),
			tags = case when $7 then $8 else tags end
		where id = $1
	`, member.Id, name, membershipStart, updateMembershipEnd, membershipEnd, ageCategory, updateTags, tags)
	if err != nil {
		return nil, err
	}

	return p.GetMember(ctx, member.Id)
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
		member          = &pb.Member{}
		membershipStart time.Time
		membershipEnd   sql.Null[time.Time]
		ageCategory     string
	)

	m := pgtype.NewMap()

	err := in.Scan(
		&member.Id,
		&member.Name,
		&membershipStart,
		&membershipEnd,
		&ageCategory,
		m.SQLScanner(&member.Tags),
	)
	if err != nil {
		return nil, err
	}

	member.MembershipStart = timestamppb.New(membershipStart)
	member.AgeCategory = pb.AgeCategory(pb.AgeCategory_value[ageCategory])

	if membershipEnd.Valid {
		member.MembershipEnd = timestamppb.New(membershipEnd.V)
	}

	return member, nil
}
