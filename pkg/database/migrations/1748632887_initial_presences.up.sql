create table presences
(
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id        uuid REFERENCES members(id),
    checkin_time     timestamp NOT NULL,
    checkout_time    timestamp
);

create index idx_presences_member_id on presences (member_id);
create index idx_presences_checkin_time on presences (checkin_time);
