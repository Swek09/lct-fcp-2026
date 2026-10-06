-- +goose Up
ALTER TABLE profiles ADD COLUMN avatar_version bigint NOT NULL DEFAULT 0;
ALTER TABLE companies ADD COLUMN avatar_version bigint NOT NULL DEFAULT 0;
ALTER TABLE companies ADD COLUMN details jsonb NOT NULL DEFAULT '{}';
CREATE TABLE account_contacts (
 user_id text PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 phone text NOT NULL DEFAULT '',
 telegram text NOT NULL DEFAULT ''
);
CREATE TABLE avatars (
 user_id text PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 image bytea NOT NULL CHECK (octet_length(image) BETWEEN 1 AND 262144)
);

-- +goose Down
DROP TABLE avatars;
DROP TABLE account_contacts;
ALTER TABLE companies DROP COLUMN details;
ALTER TABLE companies DROP COLUMN avatar_version;
ALTER TABLE profiles DROP COLUMN avatar_version;
