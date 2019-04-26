package queue

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

const (
	amqpPattern = "amqp://%s:%s@%s:%s/"
)

var conn *amqp.Connection
var Channel *amqp.Channel

func Connect(dbUser, dbPass, dbHost, dbPort string) error {
	var err error
	_, err = strconv.ParseInt(dbPort, 10, 16)
	if err != nil {
		return errors.Wrap(err, "port int parse error")
	}

	conn, err = amqp.Dial(fmt.Sprintf(amqpPattern, dbUser, dbPass, dbHost, dbPort))
	if err != nil {
		return errors.Wrap(err, "failed to connect to RabbitMQ")
	}

	Channel, err = conn.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open a channel")
	}

	return nil
}

func Close() error {
	if err := Channel.Close(); err != nil {
		return err
	}

	return conn.Close()
}
