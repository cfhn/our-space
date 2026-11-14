package presence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/cfhn/our-space/ourspace-backend/proto"
	"github.com/cfhn/our-space/pkg/database"
)

//nolint:gochecknoglobals // static lookup map
var presenceFields = map[pb.PresenceField]string{
	pb.PresenceField_PRESENCE_FIELD_ID:            "presences.id",
	pb.PresenceField_PRESENCE_FIELD_MEMBER_ID:     "presences.member_id",
	pb.PresenceField_PRESENCE_FIELD_CHECKIN_TIME:  "presences.checkin_time",
	pb.PresenceField_PRESENCE_FIELD_CHECKOUT_TIME: "presences.checkout_time",
}

var ErrDuplicate = fmt.Errorf("presence id already exists")

type Postgres struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) CreatePresence(ctx context.Context, presence *pb.Presence) (*pb.Presence, error) {
	var checkoutTime sql.Null[time.Time]

	if presence.CheckoutTime != nil {
		checkoutTime = sql.Null[time.Time]{V: presence.CheckoutTime.AsTime(), Valid: true}
	}

	_, err := p.db.ExecContext(ctx, `
		insert into presences (id, member_id, checkin_time, checkout_time)
		values ($1, $2, $3, $4);
	`, presence.Id, presence.MemberId, presence.CheckinTime.AsTime(), checkoutTime)
	if database.IsUniqueViolation(err) {
		return nil, ErrDuplicate
	}

	if err != nil {
		return nil, err
	}

	return p.GetPresenceByID(ctx, presence.Id)
}

func (p *Postgres) GetActivePresence(ctx context.Context, memberID string) (*pb.Presence, error) {
	row := p.db.QueryRowContext(ctx, `
		select id, member_id, checkin_time, checkout_time from presences where member_id = $1 and checkout_time is null
	`, memberID)

	presence, err := scanPresence(row)
	if err != nil {
		return nil, err
	}

	return presence, nil
}

func (p *Postgres) GetPresenceByID(ctx context.Context, presenceID string) (*pb.Presence, error) {
	row := p.db.QueryRowContext(ctx, `select id, member_id, checkin_time, checkout_time from presences where id = $1
	`, presenceID)

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

func (p *Postgres) UpdatePresence(
	ctx context.Context, presence *pb.Presence, fieldMask *fieldmaskpb.FieldMask,
) (*pb.Presence, error) {
	var (
		memberID       sql.Null[string]
		checkinTime    sql.Null[time.Time]
		checkoutTime   sql.Null[time.Time]
		changeCheckout bool
	)

	if fieldMask != nil {
		for _, path := range fieldMask.Paths {
			switch path {
			case "member_id":
				memberID = sql.Null[string]{V: presence.MemberId, Valid: true}
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
			checkout_time = case when $5 is true then $3::timestamptz end,
			checkin_time = coalesce($2, checkin_time),
			member_id = coalesce($4, member_id)
		where id = $1
	`, presence.Id, checkinTime, checkoutTime, memberID, changeCheckout)
	if err != nil {
		return nil, err
	}

	return p.GetPresenceByID(ctx, presence.Id)
}

func (p *Postgres) CheckoutPresence(
	ctx context.Context, memberID string, checkoutTime time.Time,
) (*pb.Presence, error) {
	presence := &pb.Presence{CheckoutTime: timestamppb.New(checkoutTime), MemberId: memberID}
	mask := &fieldmaskpb.FieldMask{Paths: []string{"checkout_time"}}

	return p.UpdatePresence(ctx, presence, mask)
}

type Filters struct {
	CheckinTimeBefore  time.Time
	CheckinTimeAfter   time.Time
	CheckoutTimeBefore time.Time
	CheckoutTimeAfter  time.Time
	MemberID           string
	CheckedOut         *bool
}

func (p *Postgres) ListPresences(
	ctx context.Context, pageSize int32, token *pb.PresencePageToken, filters *Filters, sortDirection pb.SortDirection,
	sortField pb.PresenceField,
) ([]*pb.Presence, error) {
	var (
		checkinTimeBefore  = sql.Null[time.Time]{V: filters.CheckinTimeBefore, Valid: !filters.CheckinTimeBefore.IsZero()}
		checkinTimeAfter   = sql.Null[time.Time]{V: filters.CheckinTimeAfter, Valid: !filters.CheckinTimeAfter.IsZero()}
		checkoutTimeBefore = sql.Null[time.Time]{V: filters.CheckoutTimeBefore, Valid: !filters.CheckoutTimeBefore.IsZero()}
		checkoutTimeAfter  = sql.Null[time.Time]{V: filters.CheckoutTimeAfter, Valid: !filters.CheckinTimeAfter.IsZero()}
		memberID           = sql.Null[string]{V: filters.MemberID, Valid: filters.MemberID != ""}
		checkedOut         = sql.Null[bool]{V: filters.CheckedOut != nil && *filters.CheckedOut, Valid: filters.CheckedOut != nil}
	)

	values := []any{
		pageSize,
		memberID,
		checkinTimeBefore,
		checkinTimeAfter,
		checkoutTimeBefore,
		checkoutTimeAfter,
		checkedOut,
	}

	paginationCondition, paginationValues := generatePaginationQuery(token, len(values)+1)

	values = append(values, paginationValues...)

	//nolint:gosec // safe SQL building, all dynamic data is passed through a lookup map of safe values
	rows, err := p.db.QueryContext(ctx, `
		select id, member_id, checkin_time, checkout_time from presences
		where
		($2::uuid is null OR member_id = $2)
		and	($3::timestamptz is null OR checkin_time < $3)
		and ($4::timestamptz is null OR checkin_time > $4)
		and ($5::timestamptz is null OR checkout_time < $5)
		and ($6::timestamptz is null OR checkout_time > $6)
		and ($7::bool is null OR (checkout_time is not null) = $7::bool)
		`+paginationCondition+`
		order by `+getSort(sortField, sortDirection, token)+`
		limit $1
	`, values...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

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

func generatePaginationQuery(token *pb.PresencePageToken, offset int) (string, []any) {
	fields := make([]string, 0, 2)
	values := make([]any, 0, 2)
	placeholders := make([]string, 0, 2)

	fieldName, ok := presenceFields[token.Field]
	if !ok {
		return "", nil
	}

	fields = append(fields, fieldName)
	values = append(values, token.LastValue)

	if token.Field != pb.PresenceField_PRESENCE_FIELD_ID {
		fields = append(fields, "presences.id")
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

func (p *Postgres) DeletePresence(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, `delete from presences where id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}
