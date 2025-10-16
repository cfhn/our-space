package cards

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/cfhn/our-space/ourspace-backend/proto"
)

var ErrNotFound = errors.New("card not found")

var cardFields = map[pb.CardField]string{
	pb.CardField_CARD_FIELD_ID:         "id",
	pb.CardField_CARD_FIELD_VALID_FROM: "lower(validity)",
	pb.CardField_CARD_FIELD_VALID_TO:   "upper(validity)",
}

type Filters struct {
	MemberID  string
	ValidOn   time.Time
	RfidValue []byte
}

type Postgres struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) CreateCard(ctx context.Context, card *pb.Card) (*pb.Card, error) {
	_, err := p.db.ExecContext(ctx, `
		insert into cards (id, member_id, rfid_value, validity)
		values ($1, $2, $3, tstzrange($4, $5));
	`, card.Id, card.MemberId, card.RfidValue, card.ValidFrom.AsTime(), card.ValidTo.AsTime())

	if err != nil {
		return nil, err
	}

	return p.GetCard(ctx, card.Id)
}

func (p *Postgres) GetCard(ctx context.Context, id string) (*pb.Card, error) {
	row := p.db.QueryRowContext(ctx, `
		select id, member_id, rfid_value, lower(validity), upper(validity)
		from cards
		where id = $1`, id,
	)

	member, err := scanCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (p *Postgres) ListCards(
	ctx context.Context, pageSize int32, token *pb.CardPageToken, sortField pb.CardField,
	sortDirection pb.SortDirection, filters *Filters,
) ([]*pb.Card, error) {
	paginationCondition, paginationValues := generatePaginationQuery(token, 5)

	var (
		memberID  = sql.Null[string]{V: filters.MemberID, Valid: filters.MemberID != ""}
		validOn   = sql.Null[time.Time]{V: filters.ValidOn, Valid: !filters.ValidOn.IsZero()}
		rfidValue = sql.Null[[]byte]{V: filters.RfidValue, Valid: len(filters.RfidValue) != 0}
	)

	values := []any{
		pageSize,
		memberID,
		validOn,
		rfidValue,
	}

	values = append(values, paginationValues...)

	rows, err := p.db.QueryContext(ctx, `
		select id, member_id, rfid_value, lower(validity), upper(validity)
		from cards
		where
		    ($2::uuid is null OR member_id = $2)
		and ($3::timestamptz is null OR validity @> $3)
		and ($4::bytea is null OR rfid_value = $4)
		`+paginationCondition+`
		order by `+getSort(sortField, sortDirection, token)+`
		limit $1
	`, values...)
	if err != nil {
		return nil, err
	}

	cards := make([]*pb.Card, 0, pageSize)

	for rows.Next() {
		card, err := scanCard(rows)
		if err != nil {
			return nil, err
		}

		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cards, nil
}

func generatePaginationQuery(token *pb.CardPageToken, offset int) (string, []any) {
	fields := make([]string, 0, 2)
	values := make([]any, 0, 2)
	placeholders := make([]string, 0, 2)

	fieldName, ok := cardFields[token.Field]
	if !ok {
		return "", nil
	}

	fields = append(fields, fieldName)
	values = append(values, token.LastValue)

	if token.Field != pb.CardField_CARD_FIELD_ID {
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

func getSort(sortField pb.CardField, direction pb.SortDirection, token *pb.CardPageToken) string {
	if token.Field != pb.CardField_CARD_FIELD_UNKNOWN {
		sortField = token.Field
		direction = token.Direction
	}

	fieldName, ok := cardFields[sortField]
	if !ok {
		return "id"
	}

	order := " ASC"
	if direction == pb.SortDirection_SORT_DIRECTION_DESCENDING {
		order = " DESC"
	}

	return fieldName + order + ", id" + order

}

func (p *Postgres) UpdateCard(
	ctx context.Context, member *pb.Card, fieldMask *fieldmaskpb.FieldMask,
) (*pb.Card, error) {
	var (
		memberID  sql.Null[string]
		rfidValue sql.Null[[]byte]
		validFrom sql.Null[time.Time]
		validTo   sql.Null[time.Time]
	)

	for _, path := range fieldMask.Paths {
		switch path {
		case "member_id":
			memberID = sql.Null[string]{V: member.MemberId, Valid: true}
		case "rfid_value":
			rfidValue = sql.Null[[]byte]{V: member.RfidValue, Valid: true}
		case "valid_from":
			validFrom = sql.Null[time.Time]{V: member.ValidFrom.AsTime(), Valid: true}
		case "valid_to":
			validTo = sql.Null[time.Time]{V: member.ValidTo.AsTime(), Valid: true}
		}
	}

	_, err := p.db.ExecContext(ctx, `
		update cards
		set
			member_id = coalesce($2, member_id),
			rfid_value = coalesce($3, rfid_value),
			validity =
				case when $4::timestamptz is not null AND $5::timestamptz is not null then tstzrange($4, $5)
					 when $4::timestamptz is null AND $5::timestamptz is not null then tstzrange(lower(validity), $5)
					 when $4::timestamptz is not null AND $5::time is null then tstzrange($4, upper(validity))
				else validity end
		where id = $1
	`, member.Id, memberID, rfidValue, validFrom, validTo)
	if err != nil {
		return nil, err
	}

	return p.GetCard(ctx, member.Id)
}

func (p *Postgres) DeleteCard(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, `delete from cards where id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}

type scanner interface {
	Scan(values ...any) error
}

func scanCard(in scanner) (*pb.Card, error) {
	var (
		card               = &pb.Card{}
		validFrom, validTo time.Time
	)

	err := in.Scan(
		&card.Id,
		&card.MemberId,
		&card.RfidValue,
		&validFrom,
		&validTo,
	)
	if err != nil {
		return nil, err
	}

	card.ValidFrom = timestamppb.New(validFrom)
	card.ValidTo = timestamppb.New(validTo)

	return card, nil
}
