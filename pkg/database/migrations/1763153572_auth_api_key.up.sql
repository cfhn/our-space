create table api_keys
(
    id uuid primary key default gen_random_uuid(),
    api_key text not null unique,
    member_id uuid null references members(id) on delete cascade
);
