create table if not exists monitors (
    id serial primary key,
    user_id int references users(id) on delete cascade,
    url text not null,
    created_at timestamp not null default now()
)