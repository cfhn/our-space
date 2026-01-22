package presence

import (
	"context"
	"database/sql"
	"time"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Postgres struct {
	db *sql.DB
}

func (p *Postgres) CreatePresence(ctx context.Context, memberId string) (*pb.Presence, error) {
	var (
		checkinTime sql.Null[time.Time]
	)
	checkinTime = sql.Null[time.Time]{V: time.Now(), Valid: true}
	var presenceId = uuid.New().String()
	var checkoutTime sql.Null[time.Time]

	_, err := p.db.ExecContext(ctx, `
		insert into presences (id, member_id, checkin_time, checkout_time)
		values ($1, $2, $3, $4);
	`, presenceId, memberId, checkinTime, checkoutTime)

	if err != nil {
		return nil, err
	}

	return p.GetActivePresence(ctx, memberId)
}

func (p *Postgres) GetActivePresence(ctx context.Context, memberId string) (*pb.Presence, error) {
	row := p.db.QueryRowContext(ctx, `
		SELECT from presence WHERE member_id = $1 and checkout_time is NULL
	`, memberId)

	presence, err := scanPresence(row)
	if err != nil {
		return nil, err
	}

	return presence, nil
}

func (p *Postgres) GetPresenceByID(ctx context.Context, presenceId string) (*pb.Presence, error) {
	row := p.db.QueryRowContext(ctx, `SELECT from presence WHERE member_id = $1 and checkout_time is NULL
	`, presenceId)

	presence, err := scanPresence(row)
	if err != nil {
		return nil, err
	}

	return presence, nil
}

type scanner interface {
	Scan(values ...any) error
}

func scanPresence(in scanner) (*pb.Presence, error) {
	var (
		presence     = &pb.Presence{}
		checkinTime  time.Time
		checkoutTime sql.Null[time.Time]
	)

	err := in.Scan(
		&presence.Id,
		&checkinTime,
		&checkoutTime,
	)

	if err != nil {
		return nil, err
	}

	presence.CheckinTime = timestamppb.New(checkinTime)
	presence.CheckoutTime = timestamppb.New(checkoutTime.V)

	return presence, nil
}

func (p *Postgres) UpdatePresence(ctx context.Context, presence *pb.Presence, fieldMask *fieldmaskpb.FieldMask) (*pb.Presence, error) {
	var (
		memberId     string
		checkinTime  sql.Null[time.Time]
		checkoutTime sql.Null[time.Time]
	)
	/*
		Fieldmask contains Updated Informtions
		Fieldmask has to be Validated by the Service
		Switch Case Checks for validated Values
		DB changes the given informations in given Presence
		cases: checkin_time, checkout, member_id(?),
	*/
	if fieldMask != nil {
		for i, path := range fieldMask.Paths {
			switch path {
			case "member_id":
				memberId = fieldMask.GetPaths()[i+1]
			case "checkin_time":
				checkinTime = sql.Null[time.Time]{V: presence.CheckinTime.AsTime(), Valid: true}
			case "checkout_time":
				checkoutTime = sql.Null[time.Time]{V: presence.CheckoutTime.AsTime(), Valid: true}
			}
		}
	}

	_, err := p.db.ExecContext(ctx, `
		update presences
		set,
			checkin_time = coalesce($2, checkin_time),
			checkout_time = coalesce($3, checkout_time),
			member_id = coalesce($4, member_id)
		where id = $1
	`, presence.Id, checkinTime, checkoutTime, memberId)
	if err != nil {
		return nil, err
	}

	return p.GetPresenceByID(ctx, presence.Id)
}

func (p *Postgres) CheckoutPresence(ctx context.Context, memberId string) (*pb.Presence, error) {

	mask, err := fieldmaskpb.New(&pb.UpdatePresenceRequest{}, "checkout_Time")
	if err != nil {
		return nil, err
	}
	presence, err := p.GetActivePresence(ctx, memberId)
	if err != nil {
		return nil, err
	}
	presence.CheckoutTime = timestamppb.Now()
	return p.UpdatePresence(ctx, presence, mask)

}
