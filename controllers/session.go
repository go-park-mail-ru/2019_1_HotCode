package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"

	"github.com/go-park-mail-ru/2019_1_HotCode/models"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/pkg/errors"
)

// SessionPayload структура, которая хранится в session storage
type SessionPayload struct {
	ID     int64 `json:"id"`
	PwdVer int64 `json:"pwd_ver"`
}

// CreateSession вход + кука
func CreateSession(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "SignInUser")
	errWriter := NewErrorResponseWriter(w, logger)

	form := &FormUser{}
	err := utils.DecodeBodyJSON(r.Body, form)
	if err != nil {
		errWriter.WriteWarn(http.StatusBadRequest, errors.Wrap(err, "decode body error"))
		return
	}

	session, err := createSessionImpl(form)
	if err != nil {
		if validErr, ok := err.(*ValidationError); ok {
			errWriter.WriteValidationError(validErr)
			return
		}

		errWriter.WriteError(http.StatusInternalServerError, err)
		return
	}

	// ставим куку
	http.SetCookie(w, &http.Cookie{
		Name:     "JSESSIONID",
		Value:    session.Token,
		Expires:  time.Now().Add(2628000 * time.Second),
		HttpOnly: true,
	})

	w.WriteHeader(http.StatusOK)
}

func createSessionImpl(form *FormUser) (*models.Session, error) {
	if err := form.Validate(); err != nil {
		return nil, err
	}

	user, err := models.Users.GetUserByUsername(form.Username)
	if err != nil {
		return nil, &ValidationError{
			"username": models.ErrNotExists.Error(),
		}
	}

	// пользователь удалён
	if !user.Active.Bool {
		return nil, &ValidationError{
			"username": models.ErrNotExists.Error(),
		}
	}

	if !models.Users.CheckPassword(user, form.Password) {
		return nil, &ValidationError{
			"password": models.ErrInvalid.Error(),
		}
	}

	data, err := json.Marshal(&SessionPayload{
		ID:     user.ID.Int,
		PwdVer: user.PwdVer.Int,
	})
	if err != nil {
		return nil, errors.Wrap(err, "info marshal error")
	}

	session := &models.Session{
		Payload:      data,
		ExpiresAfter: time.Hour * 24 * 30,
	}
	err = models.Sessions.Set(session)
	if err != nil {
		return nil, errors.Wrap(err, "set session error")
	}

	return session, nil
}

// DeleteSession выход + удаление куки
func DeleteSession(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "SignOutUser")
	errWriter := NewErrorResponseWriter(w, logger)

	cookie, err := r.Cookie("JSESSIONID")
	if err != nil {
		errWriter.WriteWarn(http.StatusUnauthorized, errors.Wrap(err, "get cookie error"))
		return
	}

	session := &models.Session{
		Token: cookie.Value,
	}
	err = models.Sessions.Delete(session)
	if err != nil {
		errWriter.WriteWarn(http.StatusInternalServerError, errors.Wrap(err, "session delete error"))
		return
	}

	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusOK)
}

// GetSession возвращает сессмю
func GetSession(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "GetSession")
	errWriter := NewErrorResponseWriter(w, logger)
	info := UserInfo(r)

	user, err := models.Users.GetUserByID(info.ID)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			errWriter.WriteWarn(http.StatusUnauthorized, errors.Wrap(err, "user not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, err)
		}
		return
	}

	photoUUID := ""
	if user.PhotoUUID.Status == pgtype.Present {
		photoUUID = uuid.UUID(user.PhotoUUID.Bytes).String()
	}

	utils.WriteApplicationJSON(w, http.StatusOK, &InfoUser{
		ID:     user.ID.Int,
		Active: user.Active.Bool,
		BasicUser: BasicUser{
			Username:  user.Username.String,
			PhotoUUID: photoUUID, // точно знаем, что там 16 байт
		},
	})
}
