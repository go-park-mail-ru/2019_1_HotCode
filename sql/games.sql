DROP TABLE IF EXISTS "games";
CREATE TABLE "games"
(
	id bigserial not null
		constraint game_pk
			primary key,
	title varchar(32) CONSTRAINT title_empty not null check ( title <> '' ),
	CONSTRAINT uniq_title UNIQUE(title)
);