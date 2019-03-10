package models

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"

	// требования импорта для pq
	_ "github.com/lib/pq"
)

var db DB
var storage *redis.Client

const (
	psqlUniqueViolation = "23505"
)

// ModelObserver interface of database
type ModelObserver interface {
	GetUserByID(id int64) (ModelUser, error)
	GetUserByUsername(username string) (ModelUser, error)
	getUser(params map[string]interface{}) (ModelUser, error)
}

// DB stuct for work with db implements ModelObserver
type DB struct {
	conn *gorm.DB
}

// ConnectDB открывает соединение с базой
func ConnectDB(dbUser, dbPass, dbHost, dbName string) (ModelObserver, error) {
	var err error
	db.conn, err = gorm.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s", dbUser, dbPass, dbHost, dbName))
	if err != nil {
		return nil, err
	}
	// выключаем, так как у нас есть своё логирование
	db.conn.LogMode(false)
	return &db, nil
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
	return db
}

// GetStorage returns initiated storage
func GetStorage() *redis.Client {
	return storage
}
