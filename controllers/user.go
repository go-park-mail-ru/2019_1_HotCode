package controllers

import (
	"2019_1_HotCode/models"
	"2019_1_HotCode/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// ContextKey ключ для контекста реквеста
type ContextKey int

const (
	// UserInfoKey ключ, по которому в контексте
	// реквеста хранится структура юзера после валидации
	UserInfoKey ContextKey = 1
)

// UserInfo достаёт инфу о юзере из контекста
func UserInfo(r *http.Request) *models.InfoUser {
	if rv := r.Context().Value(UserInfoKey); rv != nil {
		return rv.(*models.InfoUser)
	}
	return nil
}

// CheckUsername checks if username already used
func CheckUsername(w http.ResponseWriter, r *http.Request) {
	bUser := &models.BasicUser{}
	err := utils.DecodeBodyJSON(r.Body, bUser)
	if err != nil {
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &models.Error{
			Code:        http.StatusBadRequest,
			Description: "unable to decode request body;",
		})
		return
	}

	_, errs := models.GetUser(map[string]interface{}{
		"username": bUser.Username,
	})
	used := (errs == nil || errs.Other.Code != models.RowNotFound)
	utils.WriteApplicationJSON(w, http.StatusOK, &struct {
		Used bool `json:"used"`
	}{
		Used: used,
	})
	// log.Noticef("username %s check ok; USED: %t", bUser.Username, used)
}

// GetUser get user info by ID
func GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	//вот это всё уложить в либу
	user, errs := models.GetUser(map[string]interface{}{
		"id": vars["userID"],
	})
	if errs != nil {
		if errs.Other != nil {
			if errs.Other.Code == models.RowNotFound {
				errs.Other.Code = http.StatusNotFound
			} else if errs.Other.Code == models.InternalDatabase {
				errs.Other.Code = http.StatusInternalServerError
			}
			utils.WriteApplicationJSON(w, errs.Other.Code, errs.Other)
		}
		return
	}

	utils.WriteApplicationJSON(w, http.StatusOK, &models.InfoUser{
		ID:     user.ID,
		Active: user.Active,
		BasicUser: models.BasicUser{
			Username: user.Username,
		},
	})

	// log.Noticef("user %s was found", vars["userID"])
}

// UpdateUser updates user info by ID
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	info := UserInfo(r)

	//Попытка поменять поля без доступа к этому акку
	if vars["userID"] != strconv.Itoa(int(info.ID)) {
		utils.WriteApplicationJSON(w, http.StatusForbidden, &models.Error{
			Code:        http.StatusForbidden,
			Description: "you don't have permission to this page",
		})
		return
	}

	updateForm := &struct {
		models.BasicUser
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}{}
	err := utils.DecodeBodyJSON(r.Body, updateForm)
	if err != nil {
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &models.Error{
			Code:        http.StatusBadRequest,
			Description: "unable to decode request body;",
		})
		return
	}

	// нечего обновлять
	if updateForm.Username == "" && updateForm.NewPassword == "" {
		utils.WriteApplicationJSON(w, http.StatusOK, &models.Errors{})
		return
	}

	user, errs := models.GetUser(map[string]interface{}{
		"id": vars["userID"],
	})
	if errs != nil {
		if errs.Other != nil {
			if errs.Other.Code == models.RowNotFound {
				errs.Other.Code = http.StatusNotFound
			} else if errs.Other.Code == models.InternalDatabase {
				errs.Other.Code = http.StatusInternalServerError
			}
			utils.WriteApplicationJSON(w, errs.Other.Code, errs.Other)
		}
		return
	}

	if updateForm.Username != "" {
		user.Username = updateForm.Username
	}

	if updateForm.NewPassword != "" {
		if !user.CheckPassword(updateForm.NewPassword) {
			// log.Warningf("user: %s wrong password", user.Username)
			utils.WriteApplicationJSON(w, http.StatusBadRequest, &models.Errors{
				Fields: map[string]*models.Error{
					"oldPassword": &models.Error{
						Code:        models.WrongPassword,
						Description: "Wrong old passwrod",
					},
				},
			})
			return
		}
	}

	if errs = user.Save(); errs != nil {
		var code int
		if errs.Other != nil && errs.Other.Code == models.InternalDatabase {
			code = http.StatusInternalServerError
			errs.Other.Code = http.StatusInternalServerError
		} else {
			code = http.StatusBadRequest
		}

		utils.WriteApplicationJSON(w, code, errs)
		return
	}

	// log.Noticef("user %d updated;", info.ID)
	utils.WriteApplicationJSON(w, http.StatusOK, &models.Errors{})
}

// SignInUser signs in and returns the authentication cookie
func SignInUser(w http.ResponseWriter, r *http.Request) {
	form := &models.FormUser{}
	err := utils.DecodeBodyJSON(r.Body, form)
	if err != nil {
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &models.Error{
			Code:        http.StatusBadRequest,
			Description: "unable to decode request body;",
		})
		return
	}

	user, errs := models.GetUser(map[string]interface{}{
		"username": form.Username,
	})
	if errs != nil {
		if errs.Other != nil {
			if errs.Other.Code == models.RowNotFound {
				errs.Other.Code = http.StatusNotFound
			} else if errs.Other.Code == models.InternalDatabase {
				errs.Other.Code = http.StatusInternalServerError
			}
			utils.WriteApplicationJSON(w, errs.Other.Code, errs.Other)
		}
		return
	}

	if !user.Active {
		utils.WriteApplicationJSON(w, http.StatusNotFound, &models.Error{
			Code:        http.StatusNotFound,
			Description: "User is not active",
		})
		return
	}

	if !user.CheckPassword(form.Password) {
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &models.Errors{
			Fields: map[string]*models.Error{
				"password": &models.Error{
					Code:        models.WrongPassword,
					Description: "Wrong old passwrod",
				},
			},
		})
		return
	}

	session := models.Session{
		Info: &models.InfoUser{
			ID:     user.ID,
			Active: user.Active,
			BasicUser: models.BasicUser{
				Username: user.Username,
			},
		},
		ExpiresAfter: time.Hour * 24 * 30,
	}
	errs = session.Set()
	if errs != nil {
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, errs.Other)
		return
	}

	// ошибку можем не обрабатывать, так как
	// это сделал Set() перед нами
	bInfo, _ := json.Marshal(session.Info)

	// ставим куку
	http.SetCookie(w, &http.Cookie{
		Name:    "JSESSIONID",
		Value:   session.Token,
		Expires: time.Now().Add(2628000 * time.Second),
	})

	//уже есть готовая последовательность байт
	w.Header().Set("Content-Type", "application/json")
	w.Write(bInfo)
	// log.Noticef("username %s signin ok", user.Username)
}

// SignOutUser signs out and deletes the authentication cookie
func SignOutUser(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("JSESSIONID")
	if err != nil {
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, &models.Error{
			Code:        http.StatusInternalServerError,
			Description: "cant get cookie;",
		})
		return
	}

	session := models.Session{
		Token: cookie.Value,
	}
	errs := session.Delete()
	if errs != nil {
		utils.WriteApplicationJSON(w, http.StatusInternalServerError, errs.Other)
		return
	}

	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)

	//log.Noticef("token %s removed", cookie.Value)
	utils.WriteApplicationJSON(w, http.StatusOK, &models.Errors{})
}

// SignUpUser creates new user
func SignUpUser(w http.ResponseWriter, r *http.Request) {
	form := &models.FormUser{}
	err := utils.DecodeBodyJSON(r.Body, form)
	if err != nil {
		utils.WriteApplicationJSON(w, http.StatusBadRequest, &models.Error{
			Code:        http.StatusBadRequest,
			Description: "unable to decode request body;",
		})
		return
	}

	user := models.User{
		Username: form.Username,
		Password: form.Password,
	}

	errs := user.Create()
	if errs != nil {
		utils.WriteApplicationJSON(w, http.StatusBadRequest, errs)
		return
	}

	// log.Noticef("user %s created", user.Username)
	utils.WriteApplicationJSON(w, http.StatusOK, &models.Errors{})
}
