package users

import (
	"strconv"

	"github.com/HotCodeGroup/warscript-utils/utils"

	"github.com/go-park-mail-ru/2019_1_HotCode/database"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// UserAccessObject DAO for User model
type UserAccessObject interface {
	GetUserByID(id int64) (*UserModel, error)
	GetUserByUsername(username string) (*UserModel, error)

	Create(u *UserModel) error
	Save(u *UserModel) error
	CheckPassword(u *UserModel, password string) bool
}

// AccessObject implementation of UserAccessObject
type AccessObject struct{}

var Users UserAccessObject

func init() {
	Users = &AccessObject{}
}

// User model for users table
type UserModel struct {
	ID            pgtype.Int8
	Username      pgtype.Varchar
	PhotoUUID     pgtype.UUID
	Password      *string // строка для сохранения
	Active        pgtype.Bool
	PasswordCrypt pgtype.Bytea // внутренний хеш для проверки
	PwdVer        pgtype.Int8
}

// Create создаёт запись в базе с новыми полями
func (us *AccessObject) Create(u *UserModel) error {
	var err error
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(*u.Password), bcrypt.MinCost)
	if err != nil {
		return errors.Wrap(err, "password generate error")
	}
	u.PasswordCrypt = pgtype.Bytea{
		Bytes:  hashedPass,
		Status: pgtype.Present,
	}

	tx, err := database.Conn.Begin()
	if err != nil {
		return errors.Wrap(err, "can not open user create transaction")
	}
	defer tx.Rollback()

	_, err = us.getUserImpl(tx, "username", u.Username.String)
	if err != pgx.ErrNoRows {
		if err == nil {
			return utils.ErrTaken
		}
		return errors.Wrap(err, "select duplicate errors")
	}

	_, err = tx.Exec(`INSERT INTO users (username, password) VALUES($1, $2);`, &u.Username, &u.PasswordCrypt)
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
func (us *AccessObject) Save(u *UserModel) error {
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

	tx, err := database.Conn.Begin()
	if err != nil {
		return errors.Wrap(err, "can not open 'user Save' transaction")
	}
	defer tx.Rollback()

	du, err := us.getUserImpl(tx, "username", u.Username.String)
	if err == nil && u.ID != du.ID {
		return utils.ErrTaken
	} else if err != nil && err != pgx.ErrNoRows {
		return errors.Wrap(err, "get user save error")
	}

	_, err = tx.Exec(`UPDATE users SET (username, password, photo_uuid, active) = (
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
func (us *AccessObject) CheckPassword(u *UserModel, password string) bool {
	err := bcrypt.CompareHashAndPassword(u.PasswordCrypt.Bytes, []byte(password))
	return err == nil
}

// GetUserByID получает юзера по id
func (us *AccessObject) GetUserByID(id int64) (*UserModel, error) {
	u, err := us.getUserImpl(database.Conn, "id", strconv.FormatInt(id, 10))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, utils.ErrNotExists
		}

		return nil, errors.Wrap(err, "get user by id error")
	}

	return u, nil
}

// GetUserByUsername получает юзера по имени
func (us *AccessObject) GetUserByUsername(username string) (*UserModel, error) {
	u, err := us.getUserImpl(database.Conn, "username", username)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, utils.ErrNotExists
		}

		return nil, errors.Wrap(err, "get user by username error")
	}

	return u, nil
}

func (us *AccessObject) getUserImpl(q database.Queryer, field, value string) (*UserModel, error) {
	u := &UserModel{}

	row := q.QueryRow(`SELECT u.id, u.username, u.password,
	 					u.active, u.photo_uuid, u.pwd_ver FROM users u WHERE `+field+` = $1;`, value)
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordCrypt, &u.Active, &u.PhotoUUID, &u.PwdVer); err != nil {
		return nil, err
	}

	return u, nil
}
