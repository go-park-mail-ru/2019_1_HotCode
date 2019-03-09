package models

import (
	"github.com/lib/pq"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

// User model for users table
type User struct {
	ID        int64  `gorm:"primary_key"`
	Username  string `gorm:"size:32"`
	PhotoUUID *uuid.UUID
	// строка для сохранения(может быть задана пустой строкой)
	Password *string `gorm:"-"`
	Active   bool    `gorm:"default:true"`
	// внутренний хеш для проверки
	PasswrodCrypt []byte `gorm:"column:password"`
}

// TableName имя таблицы
func (u *User) TableName() string {
	return "users"
}

// Create создаёт запись в базе с новыми полями
func (u *User) Create() error {
	var err error
	u.PasswrodCrypt, err = bcrypt.GenerateFromPassword([]byte(*u.Password),
		bcrypt.MinCost)
	if err != nil {
		return errors.Wrap(err, "password generate error")
	}

	if err := db.Create(u).Error; err != nil {
		if pqErr, ok := err.(*pq.Error); !ok {
			return errors.Wrap(err, "user create error")
		} else if pqErr.Code == psqlUniqueViolation {
			return ErrUsernameTaken
		}

		return errors.Wrap(err, "user create error")
	}

	return nil
}

// Save сохраняет юзера в базу
func (u *User) Save() error {
	if u.Password != nil {
		newPass, err := bcrypt.GenerateFromPassword([]byte(*u.Password), bcrypt.MinCost)
		if err != nil {
			return errors.Wrap(err, "password generate error")
		}

		u.PasswrodCrypt = newPass
	}

	if err := db.Save(u).Error; err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == psqlUniqueViolation {
			return ErrUsernameTaken
		}

		return errors.Wrap(err, "user save error")
	}

	return nil
}

// CheckPassword проверяет пароль у юзера и сохранённый в модели
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword(u.PasswrodCrypt, []byte(password))
	return err == nil
}

// GetUserByUsername получает юзера по id
func GetUserByID(id int64) (*User, error) {
	return getUser(map[string]interface{}{"id": id})
}

// GetUserByUsername получает юзера по имени
func GetUserByUsername(username string) (*User, error) {
	return getUser(map[string]interface{}{"username": username})
}

func getUser(params map[string]interface{}) (*User, error) {
	u := &User{}
	if dbc := db.Where(params).First(u); dbc.Error != nil {
		if _, ok := dbc.Error.(*pq.Error); !ok &&
			(dbc.RecordNotFound() || dbc.NewRecord(u)) {
			return nil, ErrNotExists
		}

		return nil, errors.Wrap(dbc.Error, "get user error")
	}

	return u, nil
}
