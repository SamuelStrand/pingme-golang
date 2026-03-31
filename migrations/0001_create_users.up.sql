create extension if not exists pgcrypto;

create table if not exists users (
    id uuid primary key default gen_random_uuid(),
    email varchar(255) not null unique,
    password text,
    user_tg text,
    created_at timestamp not null default now()
);