DROP TABLE IF EXISTS "users" CASCADE;
create table "users"
(
	id bigserial not null
		constraint user_pk
			primary key,
	username CITEXT CONSTRAINT username_empty not null check ( username <> '' ),
	password BYTEA NOT NULL,
	active boolean default true not null,
	photo_uuid UUID DEFAULT NULL,
  CONSTRAINT unique_username UNIQUE(username)
);