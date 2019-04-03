package users

import (
	"time"

	"github.com/go-park-mail-ru/2019_1_HotCode/storage"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// SessionAccessObject DAO for Session model
type SessionAccessObject interface {
	Set(s *Session) error
	Delete(s *Session) error
	GetSession(token string) (*Session, error)
}

// SessionsDB implementation of SessionAccessObject
type Conn struct{}

var Sessions SessionAccessObject

func init() {
	Sessions = &Conn{}
}

// Session модель для работы с сессиями
type Session struct {
	Token        string
	Payload      []byte
	ExpiresAfter time.Duration
}

// Set валидирует и сохраняет сессию в хранилище по сгенерированному токену
// Токен сохраняется в s.Token
func (ss *Conn) Set(s *Session) error {
	sessionToken := uuid.NewV4()
	err := storage.Client.Set(sessionToken.String(), s.Payload, s.ExpiresAfter).Err()
	if err != nil {
		return errors.Wrap(err, "redis save error")
	}

	s.Token = sessionToken.String()
	return nil
}

// Delete удаляет сессию с токен s.Token из хранилища
func (ss *Conn) Delete(s *Session) error {
	err := storage.Client.Del(s.Token).Err()
	if err != nil {
		return errors.Wrap(err, "redis delete error")
	}

	return nil
}

// GetSession получает сессию из хранилища по токену
func (ss *Conn) GetSession(token string) (*Session, error) {
	data, err := storage.Client.Get(token).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "redis get error")
	}

	return &Session{
		Token:   token,
		Payload: data,
	}, nil
}
