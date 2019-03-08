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

	form := &FormUser{}
	err := utils.DecodeBodyJSON(r.Body, form)
	if err != nil {
		logger.Warn(errors.Wrap(err, "decode body error"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, NewAPIError(ErrBadJSON))
		return
	}

	if err = form.Validate(); err != nil {
		logger.Warn(errors.Wrap(err, "invalid form"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, err.(*ValidationError))
		return
	}

	user, err := models.GetUserByUsername(*form.Username)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			logger.Warn("username not_exists")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"username": models.ErrNotExists.Error(),
			})
		} else {
			logger.Error(errors.Wrap(err, "user get error"))
			utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err))
		}
		return
	}

	// пользователь удалён
	if !user.Active {
		logger.Warn("user was deleted")
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
			"username": models.ErrNotExists.Error(),
		})
		return
	}

	if !user.CheckPassword(*form.Password) {
		logger.Warn("password invalid")
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
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
		logger.Error(errors.Wrap(err, "info marshal error"))
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err))
		return
	}

	session := models.Session{
		Payload:      data,
		ExpiresAfter: time.Hour * 24 * 30,
	}
	err = session.Set()
	if err != nil {
		logger.Error(errors.Wrap(err, "set session error"))
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err))
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

	cookie, err := r.Cookie("JSESSIONID")
	if err != nil {
		logger.Error(errors.Wrap(err, "get cookie error"))
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(models.ErrInvalid))
		return
	}

	session := models.Session{
		Token: cookie.Value,
	}
	err = session.Delete()
	if err != nil {
		logger.Error(errors.Wrap(err, "session delete error"))
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err))
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
