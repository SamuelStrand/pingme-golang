create table if not exists telegram_link_tokens (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    token_hash text not null unique,
    expires_at timestamp not null,
    used_at timestamp null,
    created_at timestamp not null default now()
);

create index if not exists idx_telegram_link_tokens_user_id_created_at
    on telegram_link_tokens(user_id, created_at desc);

