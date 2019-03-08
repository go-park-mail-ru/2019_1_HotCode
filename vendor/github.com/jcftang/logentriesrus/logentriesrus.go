package logentriesrus

import (
	"fmt"
	"github.com/bsphere/le_go"
	"github.com/sirupsen/logrus"
	"os"
)

// Project version
const (
	VERISON = "0.0.1"
)

type LogentriesrusHook struct {
	AcceptedLevels []logrus.Level
	Client         *le_go.Logger
}

func NewLogentriesrusHook(t string) (*LogentriesrusHook, error) {
	client, err := le_go.Connect(t)
	le := LogentriesrusHook{
		Client: client,
	}
	return &le, err
}

func (l *LogentriesrusHook) Fire(e *logrus.Entry) error {
	line, err := e.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}

	l.Client.Println(line)
	return nil
}

func (l *LogentriesrusHook) Levels() []logrus.Level {
	if l.AcceptedLevels == nil {
		return AllLevels
	}
	return l.AcceptedLevels
}
