create table if not exists checklogs (
    id uuid primary key default gen_random_uuid(),
    monitor_id uuid references monitors(id) on delete cascade,
    status_code int,
    response_time_ms int,
    success boolean,
    error_message text,
    checked_at timestamp not null default now()
);