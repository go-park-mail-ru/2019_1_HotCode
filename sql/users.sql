DROP TABLE IF EXISTS "users";
create table "users"
(
	id bigserial not null
		constraint user_pk
			primary key,
	username varchar(32) CONSTRAINT username_empty not null check ( username <> '' ),
	password TEXT CONSTRAINT password_empty not null check ( password <> '' ),
	active boolean default true not null,
	photo_uuid UUID,
  CONSTRAINT uniq_username UNIQUE(username),
  CONSTRAINT uniq_photo_uuid UNIQUE (photo_uuid)
);