package briefingtypes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var ErrNotFound = errors.New("briefingType not found")

var briefingTypeFields = map[pb.BriefingTypeField]string{
	pb.BriefingTypeField_BRIEFING_TYPE_FIELD_ID: "id",
	pb.BriefingTypeField_BRIEFING_TYPE_FIELD_DISPLAY_NAME: "display_name",
	pb.BriefingTypeField_BRIEFING_TYPE_FIELD_DESCRIPTION: "description",
	pb.BriefingTypeField_BRIEFING_TYPE_FIELD_EXPIRES_AFTER: "expires_after",
}

type Filters struct {
	// todo
}

type Postgres struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func(p *Postgres) CreateBriefingType(ctx context.Context, briefingType *pb.BriefingType) (*pb.BriefingType, error) {
	_, err := p.db.ExecContext(ctx, `
		insert into briefing_types (id, displayname, description, expiresafter)
		values ($1, $2, $3, $4);
	`, briefingType.Id, briefingType.DisplayName, briefingType.Description, briefingType.ExpiresAfter)
	if err != nil {
		return nil, err
	}

	return p.GetBriefingType(ctx, briefingType.Id)
}

func (p *Postgres) GetBriefingType(ctx context.Context, id string) (*pb.BriefingType, error) {
	row := p.db.QueryRowContext(ctx,`
		select id, displayname, description, expiresafter
		from briefing_types
		where id = $1`, id,
	)

	briefingType, err := scanBriefingType(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil{
		return nil, err
	}
	return briefingType, nil
}

func (p* Postgres) DeleteBriefingType(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, `
		delete bt 
		from briefing_types where id = $1`, id) // todo when briefings exist, only delete briefingtypes without briefings
	if err != nil {
		return err
	}
	return nil
}

func getSort(sortField pb.BriefingTypeField, direction pb.SortDirection, token *pb.BriefingTypePageToken) string {
	if token.Field != pb.BriefingTypeField_BRIEFING_TYPE_FIELD_UNKNOWN {
		sortField = token.Field
		direction = token.Direction
	}

	fieldName, ok := briefingTypeFields[sortField]
	if !ok {
		return "id"
	}

	order := " ASC"
	if direction == pb.SortDirection_SORT_DIRECTION_DESCENDING {
		order = " DESC"
	}

	return fieldName + order + ", id" + order
}

func (p *Postgres) UpdateBriefingType (
	ctx context.Context, briefingType *pb.BriefingType, fieldMask *fieldmaskpb.FieldMask,
) (*pb.BriefingType, error) {
	var (
		display_name 	sql.Null[string]
		description 	sql.Null[string]
		expires_after 	sql.Null[*durationpb.Duration]
	)

	for _, path := range fieldMask.Paths {
		switch path {
		case "display_name":
			display_name = sql.Null[string] {V: briefingType.DisplayName, Valid: true}
		case "description":
			description = sql.Null[string] {V: briefingType.Description, Valid: true}
		case "expires_after":
			expires_after = sql.Null[*durationpb.Duration] {V: briefingType.ExpiresAfter, Valid: true}
		}
	}

	_, err := p.db.ExecContext(ctx, `
		update briefing_types
		set 
			briefing_type_id = coalesce($2, )
	`, display_name, description, expires_after)
	if err != nil {
		return nil, err
	}

	return p.GetBriefingType(ctx, briefingType.Id)
}

type scanner interface {
	Scan(values ...any) error
}

func scanBriefingType(in scanner) (*pb.BriefingType, error) {
	var (
		briefingType = &pb.BriefingType{}
	)

	err := in.Scan(
		&briefingType.Id,
		&briefingType.DisplayName,
		&briefingType.Description,
		&briefingType.ExpiresAfter,
	)
	if err != nil{
		return nil, err
	}

	return briefingType, nil
}

func (p* Postgres) ListBriefingTypes (
	ctx context.Context, pageSize int32, token *pb.BriefingTypePageToken, sortField pb.BriefingTypeField,
	sortDirection pb.SortDirection,
) ([]*pb.BriefingType, error) { // todo add filtering?
	var (
		
	)
	
	values := append(
		make([]any, 0, 6),
		pageSize,
	)

	paginationCondition, paginationValues := generatePaginationQuery(token, len(values)+1)
	values = append(values, paginationValues)


	rows, err := p.db.QueryContext(ctx,`
		select id, displayname, description, expiresafter
		from briefing_types
		`+paginationCondition+`
		order by `+getSort(sortField, sortDirection, token)+`
		limit $1
	`, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	briefingTypes := make([]*pb.BriefingType, 0, pageSize)

	for rows.Next(){
		briefingType, err := scanBriefingType(rows)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		briefingTypes = append(briefingTypes, briefingType)
	}

	if err != nil{
		return nil, err
	}
	return briefingTypes, nil
}


func generatePaginationQuery(token *pb.BriefingTypePageToken, offset int) (string, []any) {
	fields := make([]string, 0, 2)
	values := make([]any, 0, 2)
	placeholders := make([]string, 0, 2)

	fieldName, ok := briefingTypeFields[token.Field]
	if !ok {
		return "", nil
	}

	fields = append(fields, fieldName)
	values = append(values, token.LastValue)

	if token.Field != pb.BriefingTypeField_BRIEFING_TYPE_FIELD_ID {
		fields = append(fields, "id")
		values = append(values, token.LastId)
	}

	for i := range len(fields) {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+offset))
	}

	sort := ">"
	if token.Direction == pb.SortDirection_SORT_DIRECTION_DESCENDING{
		sort = "<"
	}

	return "and (" + strings.Join(fields, ",") + ")" + 
		sort + "(" + strings.Join(placeholders, ",") + ")", values
}