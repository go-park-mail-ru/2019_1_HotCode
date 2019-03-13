package models

import (
	"strconv"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"

	"github.com/jackc/pgx"
)

var db DB
var storage *redis.Client

// Users used for all operations on users
var Users UserAccessObject

// Games used for all operations on games
var Games GameAccessObject

// DB stuct for work with db implements ModelObserver
type DB struct {
	conn *pgx.Conn
}

type queryer interface {
	QueryRow(string, ...interface{}) *pgx.Row
}

// ConnectDB открывает соединение с базой
func ConnectDB(dbUser, dbPass, dbHost, dbPort, dbName string) error {
	var err error
	port, err := strconv.ParseInt(dbPort, 10, 16)
	if err != nil {
		return errors.Wrap(err, "port int parse error")
	}

	db.conn, err = pgx.Connect(pgx.ConnConfig{
		Host:     dbHost,
		User:     dbUser,
		Password: dbPass,
		Database: dbName,
		Port:     uint16(port),
	})
	if err != nil {
		return err
	}

	Users = &UsersDB{}
	Games = &GamesDB{}
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
