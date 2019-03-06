package controllers

import (
	"2019_1_HotCode/models"
	"2019_1_HotCode/utils"
	"net/http"
	"os"

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
		if rInfo, ok := rv.(*InfoUser); ok {
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

	bUser := &BasicUser{}
	err := utils.DecodeBodyJSON(r.Body, bUser)
	if err != nil {
		logger.Warn(errors.Wrap(err, "decode body error"))
		utils.WriteApplicationJSON(w, http.StatusBadRequest, NewAPIError(ErrBadJSON))
		return
	}

	_, err = models.GetUser(map[string]interface{}{
		"username": bUser.Username,
	}) // если база лежит
	if err != nil && errors.Cause(err) != models.ErrNotExists {
		logger.Error(errors.Wrap(err, "get user method error"))
		utils.WriteApplicationJSON(w, http.StatusInternalServerError,
			NewAPIError(err))
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
	vars := mux.Vars(r)

	user, err := models.GetUser(map[string]interface{}{
		"id": vars["user_id"],
	})
	if err != nil {
		var code int
		if errors.Cause(err) == models.ErrNotExists {
			logger.Warn("user not_exists")
			code = http.StatusNotFound
		} else {
			logger.Error(errors.Wrap(err, "get user method error"))
			code = http.StatusInternalServerError
		}
		utils.WriteApplicationJSON(w, code, NewAPIError(err))
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
		utils.WriteApplicationJSON(w, http.StatusBadRequest, NewAPIError(ErrBadJSON))
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
		var code int
		if errors.Cause(err) == models.ErrNotExists {
			logger.Warn("user not_exists")
			code = http.StatusNotFound
		} else {
			logger.Error(errors.Wrap(err, "get user method error"))
			code = http.StatusInternalServerError
		}
		utils.WriteApplicationJSON(w, code, NewAPIError(err))
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
				"oldPassword": models.ErrRequired.Error(),
			})
			return
		}

		if !user.CheckPassword(*updateForm.OldPassword) {
			logger.Warn("old password invalid")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"oldPassword": models.ErrInvalid.Error(),
			})
			return
		}
	}

	if err := user.Save(); err != nil {
		if errors.Cause(err) == models.ErrUsernameTaken {
			logger.Warn("username taken")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"username": models.ErrTaken.Error(),
			})
		} else {
			logger.Error(errors.Wrap(err, "user save error"))
			utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err))
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
		utils.WriteApplicationJSON(w, http.StatusBadRequest, NewAPIError(ErrBadJSON))
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
		if errors.Cause(err) == models.ErrUsernameTaken {
			logger.Warn("username taken")
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &ValidationError{
				"username": models.ErrTaken.Error(),
			})
		} else {
			logger.Error(errors.Wrap(err, "user create error"))
			utils.WriteApplicationJSON(w, http.StatusInternalServerError, NewAPIError(err))
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
