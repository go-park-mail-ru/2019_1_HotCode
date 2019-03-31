package models

import (
	"strconv"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	// драйвер Database
)

// UserAccessObject DAO for User model
type UserAccessObject interface {
	GetUserByID(id int64) (*User, error)
	GetUserByUsername(username string) (*User, error)

	Create(u *User) error
	Save(u *User) error
	CheckPassword(u *User, password string) bool
}

// UsersDB implementation of UserAccessObject
type UsersDB struct{}

// User model for users table
type User struct {
	ID            pgtype.Int8
	Username      pgtype.Varchar
	PhotoUUID     pgtype.UUID
	Password      *string // строка для сохранения
	Active        pgtype.Bool
	PasswordCrypt pgtype.Bytea // внутренний хеш для проверки
}

// TableName Имя таблицы
func (u *User) TableName() string {
	return "users"
}

// Create создаёт запись в базе с новыми полями
func (us *UsersDB) Create(u *User) error {
	var err error
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(*u.Password), bcrypt.MinCost)
	if err != nil {
		return errors.Wrap(err, "password generate error")
	}
	u.PasswordCrypt = pgtype.Bytea{
		Bytes:  hashedPass,
		Status: pgtype.Present,
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return errors.Wrap(err, "can not open user create transaction")
	}
	defer tx.Rollback()

	_, err = us.getUserImpl(tx, "username", u.Username.String)
	if err != pgx.ErrNoRows {
		if err == nil {
			return ErrUsernameTaken
		}
		return errors.Wrap(err, "select duplicate errors")
	}

	_, err = tx.Exec(`INSERT INTO `+u.TableName()+` (username, password) VALUES($1, $2);`, &u.Username, &u.PasswordCrypt)
	if err != nil {
		return errors.Wrap(err, "user create error")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "user create transaction commit error")
	}

	return nil
}

// Save сохраняет юзера в базу
func (us *UsersDB) Save(u *User) error {
	if u.Password != nil {
		newPass, err := bcrypt.GenerateFromPassword([]byte(*u.Password), bcrypt.MinCost)
		if err != nil {
			return errors.Wrap(err, "password generate error")
		}
		u.PasswordCrypt = pgtype.Bytea{
			Bytes:  newPass,
			Status: pgtype.Present,
		}
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return errors.Wrap(err, "can not open 'user Save' transaction")
	}
	defer tx.Rollback()

	du, err := us.getUserImpl(tx, "username", u.Username.String)
	if err == nil && u.ID != du.ID {
		return ErrUsernameTaken
	} else if err != nil && err != pgx.ErrNoRows {
		return errors.Wrap(err, "get user save error")
	}

	_, err = tx.Exec(`UPDATE `+u.TableName()+` SET (username, password, photo_uuid, active) = (
		COALESCE($1, username),
		COALESCE($2, password),
		$3,
		COALESCE($4, active)
		)
		WHERE id = $5;`,
		&u.Username, &u.PasswordCrypt, &u.PhotoUUID, &u.Active, &u.ID)
	if err != nil {
		return errors.Wrap(err, "user save error")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "user save transaction commit error")
	}

	return nil
}

// CheckPassword проверяет пароль у юзера и сохранённый в модели
func (us *UsersDB) CheckPassword(u *User, password string) bool {
	err := bcrypt.CompareHashAndPassword(u.PasswordCrypt.Bytes, []byte(password))
	return err == nil
}

// GetUserByID получает юзера по id
func (us *UsersDB) GetUserByID(id int64) (*User, error) {
	u, err := us.getUserImpl(db.conn, "id", strconv.FormatInt(id, 10))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotExists
		}

		return nil, errors.Wrap(err, "get user by id error")
	}

	return u, nil
}

// GetUserByUsername получает юзера по имени
func (us *UsersDB) GetUserByUsername(username string) (*User, error) {
	u, err := us.getUserImpl(db.conn, "username", username)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotExists
		}

		return nil, errors.Wrap(err, "get user by username error")
	}

	return u, nil
}

func (us *UsersDB) getUserImpl(q queryer, field, value string) (*User, error) {
	u := &User{}

	row := q.QueryRow(`SELECT * FROM `+u.TableName()+` WHERE `+field+` = $1;`, value)
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordCrypt, &u.Active, &u.PhotoUUID); err != nil {
		return nil, err
	}

	return u, nil
}
