package presence

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
)

var presenceFields = map[pb.PresenceField]string{
	pb.PresenceField_PRESENCE_FIELD_ID:            "presence.id",
	pb.PresenceField_PRESENCE_FIELD_MEMBER_ID:     "presence.memberId",
	pb.PresenceField_PRESENCE_FIELD_CHECKIN_TIME:  "presence.checkinTime",
	pb.PresenceField_PRESENCE_FIELD_CHECKOUT_TIME: "presence.checkoutTime",
}

type Postgres struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) CreatePresence(ctx context.Context, memberId string) (*pb.Presence, error) {
	var (
		checkinTime time.Time
	)
	checkinTime = time.Now()
	presenceId := uuid.New().String()

	_, err := p.db.ExecContext(ctx, `
		insert into presences (id, member_id, checkin_time)
		values ($1, $2, $3);
	`, presenceId, memberId, checkinTime)

	if err != nil {
		return nil, err
	}

	return p.GetActivePresence(ctx, memberId)
}

func (p *Postgres) GetActivePresence(ctx context.Context, memberId string) (*pb.Presence, error) {
	row := p.db.QueryRowContext(ctx, `
		select id, member_id, checkin_time, checkout_time from presences where member_id = $1 and checkout_time is null
	`, memberId)

	presence, err := scanPresence(row)
	if err != nil {
		return nil, err
	}

	return presence, nil
}

func (p *Postgres) GetPresenceByID(ctx context.Context, presenceId string) (*pb.Presence, error) {
	row := p.db.QueryRowContext(ctx, `select id, member_id, checkin_time, checkout_time from presences where id = $1
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
		&presence.MemberId,
		&checkinTime,
		&checkoutTime,
	)

	if err != nil {
		return nil, err
	}

	presence.CheckinTime = timestamppb.New(checkinTime)
	if checkoutTime.Valid {
		presence.CheckoutTime = timestamppb.New(checkoutTime.V)
	}

	return presence, nil
}

func (p *Postgres) UpdatePresence(ctx context.Context, presence *pb.Presence, fieldMask *fieldmaskpb.FieldMask) (*pb.Presence, error) {
	var (
		memberId sql.Null[string]
		//memberId       string
		checkinTime    sql.Null[time.Time]
		checkoutTime   sql.Null[time.Time]
		changeCheckout bool
	)

	if fieldMask != nil {
		for _, path := range fieldMask.Paths {
			switch path {
			case "member_id":
				memberId = sql.Null[string]{V: presence.MemberId, Valid: true}
				//memberId = presence.MemberId
			case "checkin_time":
				checkinTime = sql.Null[time.Time]{V: presence.CheckinTime.AsTime(), Valid: true}
			case "checkout_time":
				checkoutTime = sql.Null[time.Time]{V: presence.CheckoutTime.AsTime(), Valid: true}
				changeCheckout = true
			}
		}
	}

	_, err := p.db.ExecContext(ctx, `
		update presences
		set 
			checkout_time = case when $5 is true then cast($3 as timestamp) end,
			checkin_time = coalesce($2, checkin_time),
			member_id = coalesce($4, member_id)
		where id = $1
	`, presence.Id, checkinTime, checkoutTime, memberId, changeCheckout)
	if err != nil {
		return nil, err
	}

	return p.GetPresenceByID(ctx, presence.Id)
}

func (p *Postgres) CheckoutPresence(ctx context.Context, memberId string) (*pb.Presence, error) {

	checkoutTime := time.Now()

	row := p.db.QueryRowContext(ctx, `
		update presences
		set
			checkout_time = $2
		where member_id = $1 and checkout_time is null
		returning id, member_id, checkin_time, checkout_time
	`, memberId, checkoutTime)

	presence, err := scanPresence(row)
	if err != nil {
		return nil, err
	}
	return presence, nil

}

type Filters struct {
	CheckinTimeBefore  time.Time
	CheckinTimeAfter   time.Time
	CheckoutTimeBefore time.Time
	CheckoutTimeAfter  time.Time
	MemberId           string
}

func (p *Postgres) ListPresences(
	ctx context.Context, pageSize int32, token *pb.PresencePageToken,
	sortDirection pb.SortDirection, filters *Filters, sortField pb.PresenceField) ([]*pb.Presence, error) {
	var (
		CheckinTimeBefore  = sql.Null[time.Time]{V: filters.CheckinTimeBefore, Valid: !filters.CheckinTimeBefore.IsZero()}
		CheckinTimeAfter   = sql.Null[time.Time]{V: filters.CheckinTimeAfter, Valid: !filters.CheckinTimeAfter.IsZero()}
		CheckoutTimeBefore = sql.Null[time.Time]{V: filters.CheckoutTimeBefore, Valid: !filters.CheckoutTimeBefore.IsZero()}
		CheckoutTimeAfter  = sql.Null[time.Time]{V: filters.CheckoutTimeAfter, Valid: !filters.CheckinTimeAfter.IsZero()}
		MemberId           = sql.Null[string]{V: filters.MemberId, Valid: filters.MemberId != ""}
	)
	values := []any{
		MemberId,
		CheckinTimeBefore,
		CheckinTimeAfter,
		CheckoutTimeBefore,
		CheckoutTimeAfter,
		pageSize,
	}
	rows, err := p.db.QueryContext(ctx, `
		select id, member_id, checkin_time, checkout_time from presences
		where			    
		($1::uuid is null OR member_id = $1) 
		and	($2::timestamptz is null OR checkin_time < $2)
		and ($3::timestamptz is null OR checkin_time > $3)
		and ($4::timestamptz is null OR checkout_time < $4)
		and ($5::timestamptz is null OR checkout_time > $5)
		order by `+getSort(sortField, sortDirection, token)+`
		limit $6
	`, values...)

	if err != nil {
		return nil, err
	}

	presences := make([]*pb.Presence, 0, pageSize)

	for rows.Next() {
		presence, err := scanPresence(rows)
		if err != nil {
			return nil, err
		}

		presences = append(presences, presence)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return presences, nil
}
func getSort(sortField pb.PresenceField, direction pb.SortDirection, token *pb.PresencePageToken) string {
	if token.Field != pb.PresenceField_PRESENCE_FIELD_UNKNOWN {
		sortField = token.Field
		direction = token.Direction
	}

	fieldName, ok := presenceFields[sortField]
	if !ok {
		return "id"
	}

	order := " ASC"
	if direction == pb.SortDirection_SORT_DIRECTION_DESCENDING {
		order = " DESC"
	}

	return fieldName + order + ", id" + order
}

func (p *Postgres) DeletePresence(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, `delete from presences where id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}
