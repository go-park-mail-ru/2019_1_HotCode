package models

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// BasicUser базовые поля
type BasicUser struct {
	Username string `json:"username"`
}

// InfoUser BasicUser, расширенный служебной инфой
type InfoUser struct {
	BasicUser
	ID     int64 `json:"id"`
	Active bool  `json:"active"`
}

// FormUser BasicUser, расширенный паролем, используется для входа и регистрации
type FormUser struct {
	BasicUser
	Password string `json:"password"`
}

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

//Validate validates user parameters
func (u *User) Validate() *Errors {
	errs := Errors{
		Fields: make(map[string]*Error),
	}

	if u.Username == "" {
		errs.Fields["username"] = &Error{
			Code:        FailedToValidate,
			Description: "Username is empty",
		}
	}

	if len(errs.Fields) != 0 {
		return &errs
	}

	return nil
}

//Create validates and in case of success creates user in database
func (u *User) Create() *Errors {
	err := u.Validate()
	if err != nil {
		return err
	}
	if !db.NewRecord(u) {
		return &Errors{
			Fields: map[string]*Error{
				"id": &Error{
					Code:        CantCreate,
					Description: "User cant be formed, maybe try Update?",
				},
			},
		}
	}

	var cryptErr error
	u.PasswrodCrypt, cryptErr = bcrypt.GenerateFromPassword([]byte(u.Password),
		bcrypt.MinCost)
	if cryptErr != nil {
		return &Errors{
			Other: &Error{
				Code:        PasswordCrypt,
				Description: "cant generate hash from password",
			},
		}
	}
	u.Password = ""

	if errDB := db.Create(u).Error; errDB != nil {
		if strings.Index(errDB.Error(), "uniq_username") != -1 {
			return &Errors{
				Fields: map[string]*Error{
					"username": &Error{
						Code:        AlreadyUsed,
						Description: errDB.Error(),
					},
				},
			}
		}
		return &Errors{
			Other: &Error{
				Code:        InternalDatabase,
				Description: errDB.Error(),
			},
		}
	}
	return nil
}

//Save validates and in case of success saves user to database
func (u *User) Save() *Errors {
	err := u.Validate()
	if err != nil {
		return err
	}
	if db.NewRecord(u) {
		return &Errors{
			Fields: map[string]*Error{
				"id": &Error{
					Code:        CantSave,
					Description: "User doesn't exist, maybe try Create?",
				},
			},
		}
	}

	if u.Password != "" {
		newPass, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.MinCost)
		if err != nil {
			return &Errors{
				Other: &Error{
					Code:        PasswordCrypt,
					Description: "cant generate hash from password",
				},
			}
		}

		u.PasswrodCrypt = newPass
		u.Password = ""
	}

	if errDB := db.Save(u).Error; errDB != nil {
		if strings.Index(errDB.Error(), "uniq_username") != -1 {
			return &Errors{
				Fields: map[string]*Error{
					"username": &Error{
						Code:        AlreadyUsed,
						Description: errDB.Error(),
					},
				},
			}
		}
		return &Errors{
			Other: &Error{
				Code:        InternalDatabase,
				Description: errDB.Error(),
			},
		}
	}
	return nil
}

// CheckPassword проверяет пароль у юзера и сохранённый в модели
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword(u.PasswrodCrypt, []byte(password))
	return err != nil
}

//GetUser gets user by params
func GetUser(params map[string]interface{}) (*User, *Errors) {
	u := &User{}
	// если уедет база
	if dbc := db.Where(params).First(u); dbc.RecordNotFound() ||
		dbc.NewRecord(u) {
		return nil, &Errors{
			Other: &Error{
				Code:        RowNotFound,
				Description: "Can't find user with given parameters",
			},
		}
	} else if dbc.Error != nil {
		return nil, &Errors{
			Other: &Error{
				Code:        InternalDatabase,
				Description: dbc.Error.Error(),
			},
		}
	}

	return u, nil
}
