package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2019_1_HotCode/models"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/pkg/errors"
)

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

	user, err := models.GetUserByUsername(form.Username)
	if err != nil {
		return nil, errors.Wrap(err, "get user error")
	}

	// пользователь удалён
	if !user.Active {
		return nil, &ValidationError{
			"username": models.ErrNotExists.Error(),
		}
	}

	if !user.CheckPassword(form.Password) {
		return nil, &ValidationError{
			"password": models.ErrInvalid.Error(),
		}
	}

	data, err := json.Marshal(&InfoUser{
		ID:     user.ID,
		Active: user.Active,
		BasicUser: BasicUser{
			Username: user.Username,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "info marshal error")
	}

	session := &models.Session{
		Payload:      data,
		ExpiresAfter: time.Hour * 24 * 30,
	}
	err = session.Set()
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
		errWriter.WriteWarn(http.StatusInternalServerError, errors.Wrap(err, "get cookie error"))
		return
	}

	session := models.Session{
		Token: cookie.Value,
	}
	err = session.Delete()
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
	info := UserInfo(r)
	utils.WriteApplicationJSON(w, http.StatusOK, info)
}
