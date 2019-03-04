package controllers

import (
	"2019_1_HotCode/models"
	"2019_1_HotCode/utils"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

// ContextKey ключ для контекста реквеста
type ContextKey int

const (
	// UserInfoKey ключ, по которому в контексте
	// реквеста хранится структура юзера после валидации
	UserInfoKey ContextKey = 1
	// RequestUUIDKey ключ, по которому в контексте храниться его уникальный ID
	RequestUUIDKey ContextKey = 2
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
}

// UserInfo достаёт инфу о юзере из контекстаs
func UserInfo(r *http.Request) *InfoUser {
	if rv := r.Context().Value(UserInfoKey); rv != nil {
		return rv.(*InfoUser)
	}
	return nil
}

// TokenInfo достаёт UUID запроса
func TokenInfo(r *http.Request) string {
	if rv := r.Context().Value(RequestUUIDKey); rv != nil {
		return rv.(string)
	}

	return ""
}

func getLogger(r *http.Request, funcName string) *log.Entry {
	token := TokenInfo(r)
	return log.WithFields(log.Fields{
		"token":  token,
		"method": funcName,
	})
}

// CheckUsername checks if username already used
func CheckUsername(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "CheckUsername")

	bUser := &BasicUser{}
	err := utils.DecodeBodyJSON(r.Body, bUser)
	if err != nil {
		logger.Warn(errors.Wrap(err, "decode body error"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, NewAPIError(BadJSON))
		return
	}

	_, err = models.GetUser(map[string]interface{}{
		"username": bUser.Username,
	})
	used := (err == nil || errors.Cause(err) != models.ErrNotExists)
	utils.WriteApplicationJSON(w, http.StatusOK, &struct {
		Used bool `json:"used"`
	}{
		Used: used,
	})
}

// GetUser get user info by ID
func GetUser(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "GetUser")
	vars := mux.Vars(r)

	user, err := models.GetUser(map[string]interface{}{
		"id": vars["user_id"],
	})
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			logger.Warn("user not_exists")
			utils.WriteApplicationJSON(w, http.StatusNotFound, NewAPIError(NotExists))
		} else {
			logger.Error(errors.Wrap(err, "get user method error"))
			utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err.Error()))
		}
		return
	}

	utils.WriteApplicationJSON(w, http.StatusOK, &InfoUser{
		ID:     &user.ID,
		Active: &user.Active,
		BasicUser: BasicUser{
			Username: &user.Username,
		},
	})
}

// UpdateUser обновляет данные пользователя
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "UpdateUser")
	info := UserInfo(r)

	updateForm := &struct {
		BasicUser
		OldPassword *string `json:"oldPassword"`
		NewPassword *string `json:"newPassword"`
	}{}
	err := utils.DecodeBodyJSON(r.Body, updateForm)
	if err != nil {
		logger.Warn(errors.Wrap(err, "decode body error"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, NewAPIError(BadJSON))
		return
	}

	// нечего обновлять
	if updateForm.Username == nil && updateForm.NewPassword == nil {
		logger.Info("nothing to update")
		w.WriteHeader(http.StatusOK)
		return
	}

	user, err := models.GetUser(map[string]interface{}{
		"id": info.ID,
	})
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			logger.Warn("user not_exists")
			utils.WriteApplicationJSON(w, http.StatusNotFound, NewAPIError(NotExists))
		} else {
			logger.Error(errors.Wrap(err, "get user method error"))
			utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err.Error()))
		}
		return
	}

	if updateForm.Username != nil {
		user.Username = *updateForm.Username
	}

	// Если обновляется пароль, нужно проверить,
	// что пользователь знает старый
	if updateForm.NewPassword != nil {
		if updateForm.OldPassword == nil {
			logger.Warn("old password required")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"oldPassword": Required,
			})
			return
		}

		if !user.CheckPassword(*updateForm.OldPassword) {
			logger.Warn("old password invalid")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"oldPassword": Invalid,
			})
			return
		}
	}

	if err := user.Save(); err != nil {
		cause := errors.Cause(err)
		if cause == models.ErrUsernameTaken {
			logger.Warn("username taken")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"username": Taken,
			})
		} else {
			logger.Error(errors.Wrap(err, "user save error"))
			utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err.Error()))
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CreateUser creates new user
func CreateUser(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "CreateUser")

	form := &FormUser{}
	err := utils.DecodeBodyJSON(r.Body, form)
	if err != nil {
		logger.Warn(errors.Wrap(err, "decode body error"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, NewAPIError(BadJSON))
		return
	}

	if err := form.Validate(); err != nil {
		logger.Warn(errors.Wrap(err, "invalid form"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, err.(*ValidationError))
		return
	}

	user := models.User{
		Username: *form.Username,
		Password: *form.Password,
	}

	if err = user.Create(); err != nil {
		cause := errors.Cause(err)
		if cause == models.ErrUsernameTaken {
			logger.Warn("username taken")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"username": Taken,
			})
		} else {
			logger.Error(errors.Wrap(err, "user create error"))
			utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err.Error()))
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SignInUser signs in and returns the authentication cookie
func SignInUser(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "SignInUser")

	form := &FormUser{}
	err := utils.DecodeBodyJSON(r.Body, form)
	if err != nil {
		logger.Warn(errors.Wrap(err, "decode body error"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, NewAPIError(BadJSON))
		return
	}

	if err := form.Validate(); err != nil {
		logger.Warn(errors.Wrap(err, "invalid form"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, err.(*ValidationError))
		return
	}

	user, err := models.GetUser(map[string]interface{}{
		"username": form.Username,
	})
	if err != nil {
		cause := errors.Cause(err)
		if cause == models.ErrNotExists {
			logger.Warn("username not_exists")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"username": NotExists,
			})
		} else {
			logger.Error(errors.Wrap(err, "user get error"))
			utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err.Error()))
		}
		return
	}

	// пользователь удалён
	if !user.Active {
		logger.Warn("user was deleted")
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
			"username": NotExists,
		})
		return
	}

	if !user.CheckPassword(*form.Password) {
		logger.Warn("password invalid")
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
			"password": Invalid,
		})
		return
	}

	data, _ := json.Marshal(&InfoUser{
		ID:     &user.ID,
		Active: &user.Active,
		BasicUser: BasicUser{
			Username: &user.Username,
		},
	})
	session := models.Session{
		Payload:      data,
		ExpiresAfter: time.Hour * 24 * 30,
	}
	err = session.Set()
	if err != nil {
		logger.Error(errors.Wrap(err, "set session error"))
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err.Error()))
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

// SignOutUser signs out and deletes the authentication cookie
func SignOutUser(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "SignOutUser")

	cookie, err := r.Cookie("JSESSIONID")
	if err != nil {
		logger.Error(errors.Wrap(err, "get cookie error"))
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(Invalid))
		return
	}

	session := models.Session{
		Token: cookie.Value,
	}
	err = session.Delete()
	if err != nil {
		logger.Error(errors.Wrap(err, "session delete error"))
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err.Error()))
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
