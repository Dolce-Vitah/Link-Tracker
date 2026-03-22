CREATE TABLE IF NOT EXISTS chats (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS links (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    last_updated TIMESTAMPTZ NULL,
    last_checked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_links_url_not_blank CHECK (btrim(url) <> '')
);

CREATE TABLE IF NOT EXISTS chat_links (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    link_id BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    filters JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chat_id, link_id),
    CONSTRAINT uq_chat_links_id_chat UNIQUE (id, chat_id),
    CONSTRAINT chk_chat_links_filters_array CHECK (jsonb_typeof(filters) = 'array')
);

CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chat_id, name),
    CONSTRAINT uq_tags_id_chat UNIQUE (id, chat_id),
    CONSTRAINT chk_tags_name_not_blank CHECK (btrim(name) <> '')
);

CREATE TABLE IF NOT EXISTS chat_link_tags (
    chat_link_id BIGINT NOT NULL REFERENCES chat_links(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    PRIMARY KEY (chat_link_id, tag_id),
    CONSTRAINT fk_clt_chatlink_chat
        FOREIGN KEY (chat_link_id, chat_id)
        REFERENCES chat_links(id, chat_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_clt_tag_chat
        FOREIGN KEY (tag_id, chat_id)
        REFERENCES tags(id, chat_id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chat_links_link_id ON chat_links (link_id);
CREATE INDEX IF NOT EXISTS idx_links_last_checked_at ON links (last_checked_at);
CREATE INDEX IF NOT EXISTS idx_links_last_updated ON links (last_updated);
CREATE INDEX IF NOT EXISTS idx_tags_chat_id_id ON tags (chat_id, id);
CREATE INDEX IF NOT EXISTS idx_chat_link_tags_chat_id ON chat_link_tags (chat_id);
