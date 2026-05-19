alter table monitors
    add column if not exists slug text unique,
    add column if not exists status_page_enabled boolean not null default false;

create unique index if not exists uq_monitors_slug
    on monitors(slug) where slug is not null;