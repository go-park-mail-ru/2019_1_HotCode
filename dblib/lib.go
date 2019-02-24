package dblib

import (
	"2019_1_HotCode/apptypes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"

	"github.com/jinzhu/gorm"
	// это пакет для работы с бд, поэтому импортим драйвер бд тут
	_ "github.com/lib/pq"
)

<<<<<<< HEAD
=======
const (
	//DSN настройки соединения
	psqlStr = "postgres://ws_user:1234@localhost/warscript_lib_db"

	// docker run -d -p 6379:6379 redis
	// docker kill $$(docker ps -q)
	// docker rm $$(docker ps -a -q)
	redisDSN = "redis://user:@localhost:6379/0"
)

>>>>>>> f7d6d7f1aca1f8608e5f353094286626ac23f23f
var db *gorm.DB
var storage redis.Conn

// ConnectDB открывает соединение с базой
func ConnectDB(dbUser, dbPass, dbHost, dbName string) {
	var err error
<<<<<<< HEAD
	db, err = gorm.Open("postgres",
		fmt.Sprintf("postgres://%s:%s@%s/%s", dbUser, dbPass, dbHost, dbName))
=======
	db, err = gorm.Open("postgres", psqlStr)
>>>>>>> f7d6d7f1aca1f8608e5f353094286626ac23f23f
	if err != nil {
		log.Fatalf("failed to connect to db; err: %s", err.Error())
	}
	// выключаем, так как у нас есть своё логирование
	db.LogMode(false)

	storage, err = redis.DialURL(redisDSN)
	if err != nil {
		log.Fatalf("cant connect to redis session storage; err: %s", err.Error())
	}
}

//GetDB returns initiated database
func GetDB() *gorm.DB {
	return db
}

//GetStorage returns initiated storage
func GetStorage() redis.Conn{
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
				Code:        apptypes.LostConnectToDB,
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
				Code:        apptypes.LostConnectToDB,
				Description: errDB.Error(),
			},
		}
	}
	return nil

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
				Code:        apptypes.LostConnectToDB,
				Description: dbc.Error.Error(),
			},
		}
	}

	return u, nil
}

//StorageSet sets key-value in storage
func StorageSet(key string, value []byte, t time.Duration) error {
	_, err := storage.Do("SETEX", key, int(t.Seconds()), value)
	return errors.WithMessage(err, "failed to save value")
}

//StorageGet gets value by key
func StorageGet(key string) ([]byte, error) {
	str, err := redis.Bytes(storage.Do("GET", key))
	return str, errors.WithMessage(err, "failed to get value")
}

//StorageDel deletes value from storage
func StorageDel(key string) error {
	_, err := storage.Do("DEL", key)
	return errors.WithMessage(err, "failed to delete value")
}
