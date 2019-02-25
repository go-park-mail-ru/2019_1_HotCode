package models

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"

	// требования импорта для pq
	_ "github.com/lib/pq"
)

var db *gorm.DB
var storage redis.Conn

// ConnectDB открывает соединение с базой
func ConnectDB(dbUser, dbPass, dbHost, dbName string) error {
	var err error
	db, err = gorm.Open("postgres",
		fmt.Sprintf("postgres://%s:%s@%s/%s", dbUser, dbPass, dbHost, dbName))
	if err != nil {
		return err
	}
	// выключаем, так как у нас есть своё логирование
	db.LogMode(false)
	return nil
}

// ConnectStorage открывает соединение с хранилищем для sessions
func ConnectStorage(storageUser, storagePass, storageHost string,
	storagePort int) error {
	var err error
	storage, err = redis.DialURL(fmt.Sprintf(
		"redis://%s:%s@%s:%d/", storageUser, storagePass, storageHost, storagePort))
	if err != nil {
		return err
	}
	return nil
}

//GetDB returns initiated database
func GetDB() *gorm.DB {
	return db
}

//GetStorage returns initiated storage
func GetStorage() redis.Conn {
	return storage
}
