package dblib

import (
	"2019_1_HotCode/apptypes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/satori/go.uuid"

	"github.com/gomodule/redigo/redis"

	"github.com/jinzhu/gorm"
	// это пакет для работы с бд, поэтому импортим драйвер бд тут
	_ "github.com/lib/pq"
)

var db *gorm.DB
var storage redis.Conn

// ConnectDB открывает соединение с базой
func ConnectDB(dbUser, dbPass, dbHost, dbName string) {
	var err error
	db, err = gorm.Open("postgres",
		fmt.Sprintf("postgres://%s:%s@%s/%s", dbUser, dbPass, dbHost, dbName))
	if err != nil {
		log.Fatalf("failed to connect to db; err: %s", err.Error())
	}
	// выключаем, так как у нас есть своё логирование
	db.LogMode(false)

}

// ConnectStorage открывает соединение с хранилищем для sessions
func ConnectStorage(storageUser, storagePass, storageHost string,
	storagePort int) {
	var err error
	storage, err = redis.DialURL(fmt.Sprintf(
		"redis://%s:%s@%s:%d/0", storageUser, storagePass, storageHost, storagePort))
	if err != nil {
		log.Fatalf("cant connect to redis session storage; err: %s", err.Error())
	}
}

//GetDB returns initiated database
func GetDB() *gorm.DB {
	return db
}

//GetStorage returns initiated storage
func GetStorage() redis.Conn {
	return storage
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

//Validate validates user parameters
func (u *User) Validate() *apptypes.Errors {
	errs := apptypes.Errors{
		Fields: make(map[string]*apptypes.Error),
	}

	if u.Username == "" {
		errs.Fields["username"] = &apptypes.Error{
			Code:        apptypes.FailedToValidate,
			Description: "Username is empty",
		}
	}

	if len(u.Password) == 0 {
		errs.Fields["password"] = &apptypes.Error{
			Code:        apptypes.FailedToValidate,
			Description: "Password is empty",
		}
	}

	if len(errs.Fields) != 0 {
		return &errs
	}

	return nil
}

//Create validates and in case of success creates user in database
func (u *User) Create() *apptypes.Errors {
	err := u.Validate()
	if err != nil {
		return err
	}
	if !db.NewRecord(u) {
		return &apptypes.Errors{
			Fields: map[string]*apptypes.Error{
				"id": &apptypes.Error{
					Code:        apptypes.CantCreate,
					Description: "User cant be formed, maybe try Update?",
				},
			},
		}
	}
	if errDB := db.Create(u).Error; errDB != nil {
		if strings.Index(errDB.Error(), "uniq_username") != -1 {
			return &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"username": &apptypes.Error{
						Code:        apptypes.AlreadyUsed,
						Description: errDB.Error(),
					},
				},
			}
		}
		return &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.InternalDatabase,
				Description: errDB.Error(),
			},
		}
	}
	return nil
}

//Save validates and in case of success saves user to database
func (u *User) Save() *apptypes.Errors {
	err := u.Validate()
	if err != nil {
		return err
	}
	if db.NewRecord(u) {
		return &apptypes.Errors{
			Fields: map[string]*apptypes.Error{
				"id": &apptypes.Error{
					Code:        apptypes.CantSave,
					Description: "User doesn't exist, maybe try Create?",
				},
			},
		}
	}
	if errDB := db.Save(u).Error; errDB != nil {
		if strings.Index(errDB.Error(), "uniq_username") != -1 {
			return &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"username": &apptypes.Error{
						Code:        apptypes.AlreadyUsed,
						Description: errDB.Error(),
					},
				},
			}
		}
		return &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.InternalDatabase,
				Description: errDB.Error(),
			},
		}
	}
	return nil

}

//GetUser gets user by params
func GetUser(params map[string]interface{}) (*User, *apptypes.Errors) {
	u := &User{}
	// если уедет база
	if dbc := db.Where(params).First(u); dbc.RecordNotFound() ||
		dbc.NewRecord(u) {
		return nil, &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.RowNotFound,
				Description: "Can't find user with given parameters",
			},
		}
	} else if dbc.Error != nil {
		return nil, &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.InternalDatabase,
				Description: dbc.Error.Error(),
			},
		}
	}

	return u, nil
}

// Session модель для работы с сессиями
type Session struct {
	Token        string
	Info         *apptypes.InfoUser
	ExpiresAfter time.Duration
}

// Validate валидация сессии
func (s *Session) Validate() *apptypes.Errors {
	errs := apptypes.Errors{
		Fields: make(map[string]*apptypes.Error),
	}

	if s.Info == nil {
		errs.Fields["info"] = &apptypes.Error{
			Code:        apptypes.FailedToValidate,
			Description: "Session info is nil",
		}
	}

	if len(errs.Fields) == 0 {
		return nil
	}

	return &errs
}

// Set валидирует и сохраняет сессию в хранилище по сгенерированному токену
// Токен сохраняется в s.Token
func (s *Session) Set() *apptypes.Errors {
	errs := s.Validate()
	if errs != nil {
		return errs
	}

	sessionToken, err := uuid.NewV4()
	if err != nil {
		return &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.CantCreate,
				Description: fmt.Sprintf("unable to generate token: %s", err.Error()),
			},
		}
	}
	sessionInfo, err := json.Marshal(s.Info)
	if err != nil {
		return &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.WrongJSON,
				Description: fmt.Sprintf("unable to marshal user info: %s", err.Error()),
			},
		}
	}

	_, err = storage.Do("SETEX", sessionToken.String(),
		int(s.ExpiresAfter.Seconds()), sessionInfo)
	if err != nil {
		return &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.InternalStorage,
				Description: err.Error(),
			},
		}
	}

	s.Token = sessionToken.String()
	return nil
}

// Delete удаляет сессию с токен s.Token из хранилища
func (s *Session) Delete() *apptypes.Errors {
	_, err := storage.Do("DEL", s.Token)
	if err != nil {
		return &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.InternalStorage,
				Description: err.Error(),
			},
		}
	}
	return nil
}

// GetSession получает сессию из хранилища по токену
func GetSession(token string) (*Session, *apptypes.Errors) {
	data, err := redis.Bytes(storage.Do("GET", token))
	if err != nil {
		return nil, &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.InternalStorage,
				Description: err.Error(),
			},
		}
	}
	userInfo := &apptypes.InfoUser{}
	err = json.Unmarshal(data, userInfo)
	if err != nil {
		return nil, &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.WrongJSON,
				Description: err.Error(),
			},
		}
	}
	return &Session{
		Token: token,
		Info:  userInfo,
	}, nil
}

//
// //StorageSet sets key-value in storage
// func StorageSet(key string, value []byte, t time.Duration) error {
// 	_, err := storage.Do("SETEX", key, int(t.Seconds()), value)
// 	return errors.WithMessage(err, "failed to save value")
// }

// //StorageGet gets value by key
// func StorageGet(key string) ([]byte, error) {
// 	str, err := redis.Bytes(storage.Do("GET", key))
// 	return str, errors.WithMessage(err, "failed to get value")
// }

// //StorageDel deletes value from storage
// func StorageDel(key string) error {
// 	_, err := storage.Do("DEL", key)
// 	return errors.WithMessage(err, "failed to delete value")
// }
