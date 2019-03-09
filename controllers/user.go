package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2019_1_HotCode/models"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

// ContextKey ключ для контекста реквеста
type ContextKey int

const (
	// UserInfoKey ключ, по которому в контексте
	// реквеста хранится структура юзера после валидации
	SessionInfoKey ContextKey = 1
	// RequestUUIDKey ключ, по которому в контексте храниться его уникальный ID
	RequestUUIDKey ContextKey = 2
)

// UserInfo достаёт инфу о юзере из контекстаs
func UserInfo(r *http.Request) *SessionPayload {
	if rv := r.Context().Value(SessionInfoKey); rv != nil {
		if rInfo, ok := rv.(*SessionPayload); ok {
			return rInfo
		}
	}

	return nil
}

// TokenInfo достаёт UUID запроса
func TokenInfo(r *http.Request) string {
	if rv := r.Context().Value(RequestUUIDKey); rv != nil {
		if token, ok := rv.(string); ok {
			return token
		}
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
	errWriter := NewErrorResponseWriter(w, logger)

	bUser := &BasicUser{}
	err := utils.DecodeBodyJSON(r.Body, bUser)
	if err != nil {
		errWriter.WriteWarn(http.StatusBadRequest, errors.Wrap(err, "decode body error"))
		return
	}

	_, err = models.GetUserByUsername(bUser.Username) // если база лежит
	if err != nil && errors.Cause(err) != models.ErrNotExists {
		errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get user method error"))
		return
	}

	// Если нет ошибки, то такой юзер точно есть, а если ошибка есть,
	// то это точно models.ErrNotExists,
	// так как остальные вышли бы раньше
	used := (err == nil)
	utils.WriteApplicationJSON(w, http.StatusOK, &struct {
		Used bool `json:"used"`
	}{
		Used: used,
	})
}

// GetUser get user info by ID
func GetUser(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "GetUser")
	errWriter := NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	userID, err := strconv.ParseInt(vars["user_id"], 10, 64)
	if err != nil {
		errWriter.WriteError(http.StatusNotFound, errors.Wrap(err, "wrong format user_id"))
		return
	}

	user, err := models.GetUserByID(userID)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "user not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get user method error"))
		}
		return
	}

	utils.WriteApplicationJSON(w, http.StatusOK, &InfoUser{
		ID:     user.ID,
		Active: user.Active,
		BasicUser: BasicUser{
			Username: user.Username,
		},
	})
}

// UpdateUser обновляет данные пользователя
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "UpdateUser")
	errWriter := NewErrorResponseWriter(w, logger)
	info := UserInfo(r)

	updateForm := &FormUserUpdate{}
	err := utils.DecodeBodyJSON(r.Body, updateForm)
	if err != nil {
		errWriter.WriteWarn(http.StatusBadRequest, errors.Wrap(err, "decode body error"))
		return
	}

	err = updateUserImpl(info, updateForm)
	if err != nil {
		if validErr, ok := err.(*ValidationError); ok {
			errWriter.WriteValidationError(validErr)
			return
		}

		if errors.Cause(err) == models.ErrNotExists {
			errWriter.WriteWarn(http.StatusUnauthorized, errors.Wrap(err, "user not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	w.WriteHeader(http.StatusOK)
}

//nolint: gocyclo
func updateUserImpl(info *SessionPayload, updateForm *FormUserUpdate) error {
	if err := updateForm.Validate(); err != nil {
		return err
	}

	// нечего обновлять
	if !updateForm.Username.IsDefined() &&
		!updateForm.NewPassword.IsDefined() &&
		!updateForm.PhotoUUID.IsDefined() {
		return nil
	}

	// взяли юзера
	user, err := models.GetUserByID(info.ID)
	if err != nil {
		return errors.Wrap(err, "get user error")
	}

	// хотим обновить username
	if updateForm.Username.IsDefined() {
		user.Username = updateForm.Username.V
	}

	if updateForm.PhotoUUID.IsDefined() {
		user.PhotoUUID = &updateForm.PhotoUUID.V
	}

	// Если обновляется пароль, нужно проверить,
	// что пользователь знает старый
	if updateForm.NewPassword.IsDefined() {
		if !updateForm.OldPassword.IsDefined() {
			return &ValidationError{
				"oldPassword": models.ErrRequired.Error(),
			}
		}

		if !user.CheckPassword(updateForm.OldPassword.V) {
			return &ValidationError{
				"oldPassword": models.ErrInvalid.Error(),
			}
		}

		user.Password = &updateForm.NewPassword.V
	}

	// пытаемся сохранить
	if err := user.Save(); err != nil {
		if errors.Cause(err) == models.ErrUsernameTaken {
			return &ValidationError{
				"username": models.ErrTaken.Error(),
			}
		}

		return errors.Wrap(err, "user save error")
	}

	return nil
}

// CreateUser creates new user
func CreateUser(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "CreateUser")
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

	user := models.User{
		Username: form.Username,
		Password: &form.Password,
	}

	if err = user.Create(); err != nil {
		if errors.Cause(err) == models.ErrUsernameTaken {
			errWriter.WriteValidationError(&ValidationError{
				"username": models.ErrTaken.Error(),
			})
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "user create error"))
		}
		return
	}

	// сразу же логиним юзера
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
