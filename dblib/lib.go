package dblib

import (
	"log"
	"strings"

	errCodes "Techno/2019_1_HotCode/errors"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

type configuration struct {
	DBUser     string `env:"db_user"`
	DBPassword string `env:"db_password"`
	DBHost     string `env:"db_host"`
	DBName     string `env:"db_name"`
}

var db *gorm.DB

func init() {
	var err error
	db, err = gorm.Open("postgres", "postgres://ws_user:1234@localhost/warscript_lib_db")
	if err != nil {
		log.Fatalf("failed to connect to db; err: %s", err.Error())
	}
	db.LogMode(false)
}

// Error структура ошибки
type Error struct {
	Code        int64  `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
}

// FormErrors struct for errors:
// (поле, ошибка)
type FormErrors struct {
	Other  []*Error          `json:"other,omitempty"`
	Errors map[string]*Error `json:"errors,omitempty"`
}

//GetDB returns initiated database
func GetDB() *gorm.DB {
	return db
}

//User model for users table
type User struct {
	ID       int64  `gorm:"primary_key"`
	Username string `gorm:"size:32"`
	Password []byte
	Active   bool `gorm:"default:true"`
}

//TableName имя таблицы
func (u *User) TableName() string {
	return "user"
}

//Create validates and in case of success creates user in database
func (u *User) Create() *FormErrors {
	err := u.Validate()
	if err != nil {
		return err
	}
	if !db.NewRecord(u) {
		return &FormErrors{
			Errors: map[string]*Error{
				"id": &Error{
					Code:        errCodes.CantCreate,
					Description: "User cant be formed, maybe try Update?",
				},
			},
		}
	}
	if errDB := db.Create(u).Error; errDB != nil {
		if strings.Index(errDB.Error(), "uniq_username") != -1 {
			return &FormErrors{
				Errors: map[string]*Error{
					"username": &Error{
						Code:        errCodes.AlreadyUsed,
						Description: errDB.Error(),
					},
				},
			}
		}
		return &FormErrors{
			Errors: map[string]*Error{
				"database": &Error{
					Code:        errCodes.LostConnectToDB,
					Description: errDB.Error(),
				},
			},
		}
	}
	return nil
}

//Save validates and in case of success saves user to database
func (u *User) Save() *FormErrors {
	err := u.Validate()
	if err != nil {
		return err
	}
	if db.NewRecord(u) {
		return &FormErrors{
			Errors: map[string]*Error{
				"id": &Error{
					Code:        errCodes.CantSave,
					Description: "User doesn't exist, maybe try Create?",
				},
			},
		}
	}
	if errDB := db.Save(u).Error; errDB != nil {
		if strings.Index(errDB.Error(), "uniq_username") != -1 {
			return &FormErrors{
				Errors: map[string]*Error{
					"username": &Error{
						Code:        errCodes.AlreadyUsed,
						Description: errDB.Error(),
					},
				},
			}
		}
		return &FormErrors{
			Errors: map[string]*Error{
				"database": &Error{
					Code:        errCodes.LostConnectToDB,
					Description: errDB.Error(),
				},
			},
		}
	}
	return nil

}

//Validate validates user parameters
func (u *User) Validate() *FormErrors {
	form := FormErrors{
		Errors: make(map[string]*Error),
	}
	if u.Username == "" {
		form.Errors["username"] = &Error{
			Code:        errCodes.FailedToValidate,
			Description: "Username is empty",
		}
	}

	if len(u.Password) == 0 {
		form.Errors["password"] = &Error{
			Code:        errCodes.FailedToValidate,
			Description: "Password is empty",
		}
	}
	if len(form.Errors) != 0 {
		return &form
	}

	return nil
}

//GetUser gets user by params
func GetUser(params map[string]interface{}) (u *User, err *FormErrors) {
	u = &User{}
	if db.Where(params).First(u).RecordNotFound() || db.NewRecord(u) {
		return nil, &FormErrors{
			Errors: map[string]*Error{
				"pattern_mismatch": {
					Code:        errCodes.RowNotFound,
					Description: "Can't find user with given parameters",
				},
			},
		}
	}
	return u, nil
}
