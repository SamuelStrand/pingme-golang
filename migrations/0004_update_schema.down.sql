alter table users
drop column if exists password;
drop column if exists user_tg;

alter table monitors
drop column if exists name,
drop column if exists interval_seconds,
drop column if exists timeout_seconds,
drop column if exists enabled,
drop column if exists last_status;

alter table checklogs
drop column if exists success,
drop column if exists error_message;