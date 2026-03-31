alter table users
drop column password text;
drop column user_tg text;

alter table monitors
drop column name text,
drop column interval_seconds,
drop column timeot_seconds,
drop column enabled boolean,
drop column last_status text;

alter table checklogs
drop column success,
drop column error_message;