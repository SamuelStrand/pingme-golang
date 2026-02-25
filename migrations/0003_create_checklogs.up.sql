create table if not exists checklogs (
    id serial primary key,
    monitor_id int references monitors(id) on delete cascade,
    status_code int,
    response_time_ms int,
    checked_at timestamp not null default now()
)