package briefings

import (
	"context"
	"database/sql"
	"errors"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
)

var ErrNotFound = errors.New("briefingType not found")


type Postgres struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func(p *Postgres) CreateBriefingType(ctx context.Context, briefingType *pb.BriefingType) (*pb.Card, error) {
	_, err := p.db.ExecContext(ctx, `
		insert into briefingtypes (id, displayname, description, expiresafter)
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


	if err != nil{
		return nil, err
	}
	return briefing
}

type scanner interface {
	Scan(values ...any) error
}

func scanBriefingType(in briefingtype) (*pb.BriefingType, error) {
	var (
		card = &pb.Card{}
	)

	err := in.Scan(
		&card.Id
	)
}