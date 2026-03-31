alter table users
add column password text;
add column user_tg text;

alter table monitors
add column name text,
add column interval_seconds int default 60,
add column timeot_seconds int default 5,
add column enabled boolean default true,
add column last_status text default 'unknown';

alter table checklogs
add column success boolean,
add column error_message text;