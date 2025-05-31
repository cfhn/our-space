package presences

import (
	"context"
	"database/sql"
	"errors"

	"github.com/cfhn/our-space/ourspace-backend/pb"
)

var presenceFields = map[pb.PresenceField]string{
	pb.PresenceField_PRESENCE_FIELD_ID:            "id",
	pb.PresenceField_PRESENCE_FIELD_MEMBER_ID:     "member_id",
	pb.PresenceField_PRESENCE_FIELD_CHECKIN_TIME:  "checkin_time",
	pb.PresenceField_PRESENCE_FIELD_CHECKOUT_TIME: "chckout_time",
}

type Postgres struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

