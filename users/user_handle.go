package users

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"

	"github.com/HotCodeGroup/warscript-utils/utils"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// CheckUsername checks if username already used
func CheckUsername(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "CheckUsername")
	errWriter := utils.NewErrorResponseWriter(w, logger)

	bUser := &BasicUser{}
	err := utils.DecodeBodyJSON(r.Body, bUser)
	if err != nil {
		errWriter.WriteWarn(http.StatusBadRequest, errors.Wrap(err, "decode body error"))
		return
	}

	_, err = Users.GetUserByUsername(bUser.Username) // если база лежит
	if err != nil && errors.Cause(err) != utils.ErrNotExists {
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
	logger := utils.GetLogger(r, "GetUser")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	userID, err := strconv.ParseInt(vars["user_id"], 10, 64)
	if err != nil {
		errWriter.WriteError(http.StatusNotFound, errors.Wrap(err, "wrong format user_id"))
		return
	}

	user, err := Users.GetUserByID(userID)
	if err != nil {
		if errors.Cause(err) == utils.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "user not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get user method error"))
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

// UpdateUser обновляет данные пользователя
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "UpdateUser")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	info := SessionInfo(r)
	if info == nil {
		errWriter.WriteWarn(http.StatusUnauthorized, errors.New("session info is not presented"))
		return
	}

	updateForm := &FormUserUpdate{}
	err := utils.DecodeBodyJSON(r.Body, updateForm)
	if err != nil {
		errWriter.WriteWarn(http.StatusBadRequest, errors.Wrap(err, "decode body error"))
		return
	}

	err = updateUserImpl(info, updateForm)
	if err != nil {
		if validErr, ok := err.(*utils.ValidationError); ok {
			errWriter.WriteValidationError(validErr)
			return
		}

		if errors.Cause(err) == utils.ErrNotExists {
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
	user, err := Users.GetUserByID(info.ID)
	if err != nil {
		return errors.Wrap(err, "get user error")
	}

	// хотим обновить username
	if updateForm.Username.IsDefined() {
		user.Username = pgtype.Varchar{
			String: updateForm.Username.V,
			Status: pgtype.Present,
		}
	}

	if updateForm.PhotoUUID.IsDefined() {
		var photoUUID uuid.UUID
		status := pgtype.Null
		if updateForm.PhotoUUID.V != "" {
			status = pgtype.Present
			photoUUID = uuid.MustParse(updateForm.PhotoUUID.V)
		}

		user.PhotoUUID = pgtype.UUID{
			Bytes:  photoUUID,
			Status: status,
		}
	}

	// Если обновляется пароль, нужно проверить,
	// что пользователь знает старый
	if updateForm.NewPassword.IsDefined() {
		if !updateForm.OldPassword.IsDefined() {
			return &utils.ValidationError{
				"oldPassword": utils.ErrRequired.Error(),
			}
		}

		if !Users.CheckPassword(user, updateForm.OldPassword.V) {
			return &utils.ValidationError{
				"oldPassword": utils.ErrInvalid.Error(),
			}
		}

		user.Password = &updateForm.NewPassword.V
	}

	// пытаемся сохранить
	if err := Users.Save(user); err != nil {
		if errors.Cause(err) == utils.ErrTaken {
			return &utils.ValidationError{
				"username": utils.ErrTaken.Error(),
			}
		}

		return errors.Wrap(err, "user save error")
	}

	return nil
}

// CreateUser creates new user
func CreateUser(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "CreateUser")
	errWriter := utils.NewErrorResponseWriter(w, logger)

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

	user := &UserModel{
		Username: pgtype.Varchar{
			String: form.Username,
			Status: pgtype.Present,
		},
		Password: &form.Password,
	}

	if err = Users.Create(user); err != nil {
		if errors.Cause(err) == utils.ErrTaken {
			errWriter.WriteValidationError(&utils.ValidationError{
				"username": utils.ErrTaken.Error(),
			})
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "user create error"))
		}
		return
	}

	// сразу же логиним юзера
	session, err := CreateSessionImpl(form)
	if err != nil {
		if validErr, ok := err.(*utils.ValidationError); ok {
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
