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

DROP FUNCTION IF EXISTS users_count_increment CASCADE;
CREATE FUNCTION users_count_increment() RETURNS TRIGGER AS $_$
BEGIN
IF NEW.password != OLD.password THEN
	UPDATE users SET pwd_ver = pwd_ver + 1 WHERE id = NEW.id;
end if;
RETURN NEW;
END $_$ LANGUAGE 'plpgsql';

CREATE TRIGGER users_insert_trigger AFTER UPDATE ON users
  FOR EACH ROW EXECUTE PROCEDURE users_count_increment();