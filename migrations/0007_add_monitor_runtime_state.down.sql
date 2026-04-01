drop index if exists idx_monitors_enabled_next_check_at;

alter table monitors
drop column if exists consecutive_failures,
drop column if exists next_check_at,
drop column if exists last_checked_at;
