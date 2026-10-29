CREATE TABLE ad_publications (
    id           BIGSERIAL   PRIMARY KEY,
    user_id      UUID        NOT NULL,
    published_at TIMESTAMPTZ NOT NULL
);

INSERT INTO ad_publications (user_id, published_at)
SELECT user_id, created_at
FROM ads;

CREATE INDEX ad_publications_user_published_at_idx
    ON ad_publications (user_id, published_at);
