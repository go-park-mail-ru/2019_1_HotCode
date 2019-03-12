package models

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"

	// требования импорта для pq
	_ "github.com/lib/pq"
)

var db DB

// Users used for all operations on users
var Users UserAccessObject
var storage *redis.Client

const (
	psqlUniqueViolation = "23505"
)

// UserAccessObject DAO for User model
type UserAccessObject interface {
	GetUserByID(id int64) (*User, error)
	GetUserByUsername(username string) (*User, error)
	getUser(params map[string]interface{}) (*User, error)

	Create(u *User) error
	Save(u *User) error
	CheckPassword(u *User, password string) bool
}

// UsersDB implementation of UserAccessObject
type UsersDB struct{}

// DB stuct for work with db implements ModelObserver
type DB struct {
	conn *gorm.DB
}

// ConnectDB открывает соединение с базой
func ConnectDB(dbUser, dbPass, dbHost, dbName string) error {
	var err error
	db.conn, err = gorm.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s", dbUser, dbPass, dbHost, dbName))
	if err != nil {
		return err
	}
	// выключаем, так как у нас есть своё логирование
	db.conn.LogMode(false)

	Users = &UsersDB{}
	return nil
}

// ConnectStorage открывает соединение с хранилищем для sessions
func ConnectStorage(storageUser, storagePass, storageHost string) error {
	var err error
	storage = redis.NewClient(&redis.Options{
		Addr:     storageHost,
		Password: storagePass,
		DB:       0,
	})
	_, err = storage.Ping().Result()
	if err != nil {
		return err
	}
	return nil
}

// GetDB returns initiated database
func GetDB() *gorm.DB {
	return db.conn
}

// GetStorage returns initiated storage
func GetStorage() *redis.Client {
	return storage
}
