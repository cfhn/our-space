alter table cards
    drop constraint cards_member_id_fkey,
    add constraint cards_member_id_fkey
        foreign key (member_id)
        references members(id)
        on delete cascade;
