DROP TYPE IF EXISTS LANG;
CREATE TYPE LANG AS ENUM ('JS');

DROP TABLE IF EXISTS "bots";
CREATE TABLE "bots"
(
	id BIGSERIAL NOT NULL
		CONSTRAINT bot_pk
			PRIMARY KEY,
	code TEXT CONSTRAINT code_empty NOT NULL CHECK ( code <> '' ),
	code_hash BYTEA NOT NULL CHECK ( code_hash <> '' ),
	language LANG NOT NULL,
	is_active BOOLEAN NOT NULL DEFAULT FALSE,
	author_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	game_id BIGINT NOT NULL REFERENCES games (id) ON DELETE CASCADE,

	CONSTRAINT unique_code UNIQUE (code_hash, language, author_id, game_id)
);