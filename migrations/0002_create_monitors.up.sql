create table if not exists monitors (
    id uuid primary key default gen_random_uuid(),
    user_id uuid references users(id) on delete cascade,
    url text not null,
    name text,
    interval_seconds int not null default 60,
    timeout_seconds int not null default 5,
    enabled boolean not null default true,
    last_status text not null default 'unknown',
    created_at timestamp not null default now()
);