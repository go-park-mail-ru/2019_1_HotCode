DROP TABLE IF EXISTS "users_games";
CREATE TABLE "users_games"
(
	user_id BIGINT REFERENCES users (id) ON DELETE CASCADE,
	game_id BIGINT REFERENCES games (id) ON DELETE CASCADE,
	score INTEGER NOT NULL DEFAULT 0,
	CONSTRAINT users_games_pk PRIMARY KEY (user_id, game_id)
);