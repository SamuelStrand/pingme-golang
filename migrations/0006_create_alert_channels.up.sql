create table if not exists alert_channels (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    type text not null,
    address text not null,
    enabled boolean not null default true,
    created_at timestamp not null default now()
);

create index if not exists idx_alert_channels_user_id on alert_channels(user_id);
create index if not exists idx_alert_channels_user_id_enabled on alert_channels(user_id, enabled);
create unique index if not exists uq_alert_channels_user_type_address on alert_channels(user_id, type, address);
