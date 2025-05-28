create table members
(
    id               uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    name             text        NOT NULL,
    membership_start timestamptz NOT NULL,
    membership_end   timestamp,
    age_category     text        NOT NULL,
    tags             text[]      NOT NULL DEFAULT ARRAY []::text[]
);

create table cards
(
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id  uuid REFERENCES members (id),
    rfid_value bytea     NOT NULL,
    validity   tstzrange NOT NULL
);

create index idx_cards_member_id on cards (member_id);
create index idx_cards_rfid_value on cards (rfid_value);
