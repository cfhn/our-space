create table member_attributes (
    id uuid primary key,
    technical_name text not null unique,
    display_name text not null,
    type text not null,
    description text
);

alter table members
    add column additional_attributes
        jsonb
        not null
        default '{}'::jsonb;
