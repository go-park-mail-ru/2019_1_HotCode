DROP TABLE IF EXISTS "games" CASCADE;
CREATE TABLE "games"
(
	id bigserial not null
		constraint game_pk
			primary key,
	title CITEXT CONSTRAINT title_empty not null check ( title <> '' ),
	CONSTRAINT unique_title UNIQUE(title)
);