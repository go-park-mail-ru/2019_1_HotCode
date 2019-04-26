package database

import (
	"strconv"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type Queryer interface {
	QueryRow(string, ...interface{}) *pgx.Row
}

var Conn *pgx.ConnPool

// Connect открывает соединение с базой
func Connect(dbUser, dbPass, dbHost, dbPort, dbName string) error {
	var err error
	port, err := strconv.ParseInt(dbPort, 10, 16)
	if err != nil {
		return errors.Wrap(err, "port int parse error")
	}

	Conn, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     dbHost,
			User:     dbUser,
			Password: dbPass,
			Database: dbName,
			Port:     uint16(port),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func Close() {
	Conn.Close()
}
