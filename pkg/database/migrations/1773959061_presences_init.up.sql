create table presences(
    id uuid primary key,
    member_id text NOT NULL,
    checkin_time timestamptz NOT NULL
    checkout_time timestamptz
);