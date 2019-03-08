package models

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"

	// требования импорта для pq
	_ "github.com/lib/pq"
)

var db *gorm.DB
var storage *redis.Client

const (
	psqlUniqueViolation = "23505"
)

// ConnectDB открывает соединение с базой
func ConnectDB(dbUser, dbPass, dbHost, dbName string) error {
	var err error
	db, err = gorm.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s", dbUser, dbPass, dbHost, dbName))
	if err != nil {
		return err
	}
	// выключаем, так как у нас есть своё логирование
	db.LogMode(false)
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
	return db
}

// GetStorage returns initiated storage
func GetStorage() *redis.Client {
	return storage
}
