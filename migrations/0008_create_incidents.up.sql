create table if not exists incidents (
    id uuid primary key default gen_random_uuid(),
    monitor_id uuid not null references monitors(id) on delete cascade,
    status text not null,
    reason text null,
    started_at timestamp not null default now(),
    resolved_at timestamp null,
    created_at timestamp not null default now()
);

create index if not exists idx_incidents_monitor_id on incidents(monitor_id);
create unique index if not exists uq_incidents_open_monitor on incidents(monitor_id) where status = 'open';
