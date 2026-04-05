create index if not exists idx_monitors_user_created_at_id
    on monitors(user_id, created_at desc, id desc);

create index if not exists idx_checklogs_monitor_checked_at_id
    on checklogs(monitor_id, checked_at desc, id desc);
