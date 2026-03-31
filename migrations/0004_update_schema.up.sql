alter table users
add column if not exists password text,
add column if not exists user_tg text;

alter table monitors
add column if not exists name text,
add column if not exists interval_seconds int not null default 60,
add column if not exists timeout_seconds int not null default 5,
add column if not exists enabled boolean not null default true,
add column if not exists last_status text not null default 'unknown';

alter table checklogs
add column if not exists success boolean,
add column if not exists error_message text;