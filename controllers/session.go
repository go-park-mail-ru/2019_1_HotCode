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

	if valError := form.Validate(); valError != nil {
		errWriter.WriteValidationError(valError)
		return
	}

	user, err := models.GetUserByUsername(*form.Username)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			errWriter.WriteValidationError(&ValidationError{
				"username": models.ErrNotExists.Error(),
			})
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "user get error"))
		}
		return
	}

	// пользователь удалён
	if !user.Active {
		errWriter.WriteValidationError(&ValidationError{
			"username": models.ErrNotExists.Error(),
		})
		return
	}

	if !user.CheckPassword(*form.Password) {
		errWriter.WriteValidationError(&ValidationError{
			"password": models.ErrInvalid.Error(),
		})
		return
	}

	data, err := json.Marshal(&InfoUser{
		ID:     &user.ID,
		Active: &user.Active,
		BasicUser: BasicUser{
			Username: &user.Username,
		},
	})
	if err != nil {
		errWriter.WriteWarn(http.StatusInternalServerError, errors.Wrap(err, "info marshal error"))
		return
	}

	session := models.Session{
		Payload:      data,
		ExpiresAfter: time.Hour * 24 * 30,
	}
	err = session.Set()
	if err != nil {
		errWriter.WriteWarn(http.StatusInternalServerError, errors.Wrap(err, "set session error"))
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
