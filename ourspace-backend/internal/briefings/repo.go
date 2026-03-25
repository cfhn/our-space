package briefings

import (
	"context"
	"database/sql"
	"errors"
	// "time"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	//"golang.org/x/text/cases"
	"google.golang.org/protobuf/types/known/durationpb"
	//"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var ErrNotFound = errors.New("briefingType not found")


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
		select id, displayname, description, expiresafter)
		`, id,
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

//func ListBriefingTypes(p* Postgres) (*pb.BriefingTyp)