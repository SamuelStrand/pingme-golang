alter table monitors
add column if not exists consecutive_failures int not null default 0,
add column if not exists next_check_at timestamp not null default now(),
add column if not exists last_checked_at timestamp null;

create index if not exists idx_monitors_enabled_next_check_at on monitors(enabled, next_check_at);
