create table if not exists users (
    id serial primary key,
    email varchar(255) not null unique,
    created_at timestamp not null default now()
)