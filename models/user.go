package models

import (
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

//User model for users table
type User struct {
	ID       int64  `gorm:"primary_key"`
	Username string `gorm:"size:32"`
	// строка для сохранения
	Password string `gorm:"-"`
	Active   bool   `gorm:"default:true"`
	// внутренний хеш для проверки
	PasswrodCrypt []byte `gorm:"column:password"`
}

//TableName имя таблицы
func (u *User) TableName() string {
	return "user"
}

//Create создаёт запись в базе с новыми полями
func (u *User) Create() error {
	var cryptErr error
	u.PasswrodCrypt, cryptErr = bcrypt.GenerateFromPassword([]byte(u.Password),
		bcrypt.MinCost)
	if cryptErr != nil {
		return errors.Wrap(cryptErr, "password generate error")
	}

	if errDB := db.Create(u).Error; errDB != nil {
		if errDB.(*pq.Error).Code == "23505" {
			return ErrUsernameTaken
		}

		return errors.Wrap(errDB, "user create error")
	}

	return nil
}

//Save сохраняет юзера в базу
func (u *User) Save() error {
	if u.Password != "" {
		newPass, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.MinCost)
		if err != nil {
			return errors.Wrap(err, "password generate error")
		}

		u.PasswrodCrypt = newPass
	}

	if errDB := db.Save(u).Error; errDB != nil {
		if errDB.(*pq.Error).Code == "23505" {
			return ErrUsernameTaken
		}

		return errors.Wrap(errDB, "user save error")
	}

	return nil
}

// CheckPassword проверяет пароль у юзера и сохранённый в модели
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword(u.PasswrodCrypt, []byte(password))
	return err == nil
}

//GetUser получает юзера по данным параметрам
func GetUser(params map[string]interface{}) (*User, error) {
	u := &User{}
	if dbc := db.Where(params).First(u); dbc.RecordNotFound() ||
		dbc.NewRecord(u) {
		return nil, ErrNotExists
	} else if dbc.Error != nil {
		return nil, errors.Wrap(dbc.Error, "get user internal error")
	}

	return u, nil
}
