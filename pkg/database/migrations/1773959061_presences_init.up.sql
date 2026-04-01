create table presences
(
    id              uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    member_id       uuid REFERENCES members (id),
    checkin_time    timestamptz NOT NULL,
    checkout_time   timestamptz
    );