package presence

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
)

type Postgres struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *Postgres {
	return &Postgres{db: db}
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
		SELECT id, member_id, checkin_time, checkout_time from presences WHERE member_id = $1 and checkout_time is NULL
	`, memberId)

	presence, err := scanPresence(row)
	if err != nil {
		return nil, err
	}

	return presence, nil
}

func (p *Postgres) GetPresenceByID(ctx context.Context, presenceId string) (*pb.Presence, error) {
	row := p.db.QueryRowContext(ctx, `SELECT id, member_id, checkin_time, checkout_time from presences WHERE id = $1
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

func (p *Postgres) UpdatePresence(ctx context.Context, presence *pb.Presence) (*pb.Presence, error) {
	var (
		memberId     string
		checkinTime  sql.Null[time.Time]
		checkoutTime sql.Null[time.Time]
	)

	// if fieldMask != nil {
	// 	for _, path := range fieldMask.Paths {
	// 		switch path {
	// case "member_id":
	memberId = presence.MemberId
	// case "checkin_time":
	checkinTime = sql.Null[time.Time]{V: presence.CheckinTime.AsTime(), Valid: true}
	if presence.CheckoutTime != nil {
		checkoutTime = sql.Null[time.Time]{V: presence.CheckoutTime.AsTime(), Valid: true}
	}
	// case "checkout_time":
	// }
	// }
	// }
	_, err := p.db.ExecContext(ctx, `
		update presences
		set
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

	presence, err := p.GetActivePresence(ctx, memberId)
	if err != nil {
		return nil, err
	}
	presence.CheckoutTime = timestamppb.Now()
	return p.UpdatePresence(ctx, presence)

}

type Filters struct {
	CheckinTimeBefore  time.Time
	CheckinTimeAfter   time.Time
	CheckoutTimeBefore time.Time
	CheckoutTimeAfter  time.Time
	MemberId           string
}

func (p *Postgres) ListPresences(
	ctx context.Context, pageSize int32, token *pb.MemberPageToken, sortDirection pb.SortDirection, filters *Filters) ([]*pb.Presence, error) {
	const offset int = 9
	//	paginationCondition, paginationValues := generatePaginationQuery(token, offset)
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
		limit $6
	`, values...)

	if err != nil {
		return nil, err
	}
	//eyJhbGciOiJFUzI1NiIsImtpZCI6InJmRGx6YjdZenZQazRPUFZtUWlrR1ZMdlBpVHNjUnR2REdxcmZDR00wVjQiLCJ0eXAiOiJKV1QifQ.eyJzdWIiOiJhZG1pbiIsImV4cCI6MTc3NTE1MTA0NCwibmJmIjoxNzc1MTUwMTI5LCJpYXQiOjE3NzUxNTAxNDQsImp0aSI6ImU0YWJlMjY1LWI5MmYtNDI5MS04Y2E3LTk3NGFjNDEyMmQzOCIsInR5cGUiOiJhY2Nlc3MiLCJmdWxsX25hbWUiOiJJbml0aWFsIFVzZXIifQ.ACrSamPJmAf2NUoDEwWtGCSJh79vdD2eGxFXavNjBFmde2Q3YHQs0UteizjaKLRaepN2t2SOBQ9zIqhsb9dDdA
	// error(*pgconn.PgError) *{Severity: "ERROR", SeverityUnlocalized: "ERROR", Code: "22P02", Message: "invalid input syntax for type uuid: \"\"", Detail: "", Hint: "", Position: 0, InternalPosition: 0, InternalQuery: "", Where: "unnamed portal parameter $1 = '...
	//error(*pgconn.PgError) *{Severity: "ERROR", SeverityUnlocalized: "ERROR", Code: "22P02", Message: "invalid input syntax for type uuid: \"\"", Detail: "", Hint: "", Position: 0, InternalPosition: 0, InternalQuery: "", Where: "unnamed portal parameter $1 = '...
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

func (p *Postgres) DeletePresence(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, `delete from presences where id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}
