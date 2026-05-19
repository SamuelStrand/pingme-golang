drop index if exists uq_monitors_slug;

alter table monitors
drop column if exists status_page_enabled,
    drop column if exists slug;