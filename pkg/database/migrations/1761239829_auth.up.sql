create table members_auth
(
    id uuid PRIMARY KEY REFERENCES members(id) on delete cascade,
    username text not null,
    password_hash text not null
)