package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	logging "github.com/op/go-logging"
	"golang.org/x/crypto/bcrypt"

	"2019_1_HotCode/apptypes"
	"2019_1_HotCode/dblib"
)

// docker run -d -p 6379:6379 redis
// docker kill $$(docker ps -q)
// docker rm $$(docker ps -a -q)

var (
	log = logging.MustGetLogger("auth")

	logFormat = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
)

// Handler пока что только хранит темплейты
// потом можно добавить grpc клиенты
type Handler struct {
	Router           http.Handler
	Tmpls            map[string]*template.Template
	DBConn           *gorm.DB
	SessionStoreConn redis.Conn
}

// Index рисует индекс
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.Tmpls["index.html"].ExecuteTemplate(w, "tmp", struct{}{})
}

// CheckToken метод проверяющий токен, возвращает инфу о юзере(без пароля)
// позже выделится в отдельный запрос
func (h *Handler) CheckToken(sessionToken string) (*apptypes.InfoUser, *apptypes.Errors) {
	session, errs := dblib.GetSession(sessionToken)
	if errs != nil {
		return nil, errs
	}

	return session.Info, nil
}

// CheckUsername checks if username already used
func (h *Handler) CheckUsername(w http.ResponseWriter, r *http.Request) {
	bUser := &apptypes.BasicUser{}
	err := decodeBodyJSON(r.Body, bUser)
	if err != nil {
		writeFatalError(w, http.StatusBadRequest,
			fmt.Sprintf("unable to decode request body; err: %s", err.Error()),
			"incorrect json")
		return
	}

	_, errs := dblib.GetUser(map[string]interface{}{
		"username": bUser.Username,
	})
	used := (errs == nil || errs.Other.Code != apptypes.RowNotFound)
	writeApplicationJSON(w, &struct {
		Used bool `json:"used"`
	}{
		Used: used,
	})

	log.Noticef("username %s check ok; USED: %t", bUser.Username, used)
}

// GetUser get user info by ID
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	//вот это всё уложить в либу
	user, errs := dblib.GetUser(map[string]interface{}{
		"id": vars["userID"],
	})
	if errs != nil {
		writeApplicationJSON(w, errs)
		return
	}

	writeApplicationJSON(w, &apptypes.InfoUser{
		ID:     user.ID,
		Active: user.Active,
		BasicUser: apptypes.BasicUser{
			Username: user.Username,
		},
	})

	log.Noticef("user %s was found", vars["userID"])
}

// UpdateUser updates user info by ID
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	info := userInfo(r)

	//Попытка поменять поля без доступа к этому акку
	if vars["userID"] != strconv.Itoa(int(info.ID)) {
		writeFatalError(w, http.StatusForbidden,
			fmt.Sprintf("%s tried to change %d", vars["userID"], info.ID),
			"you don't have permission to this page")
		return
	}

	updateForm := &struct {
		apptypes.BasicUser
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}{}
	err := decodeBodyJSON(r.Body, updateForm)
	if err != nil {
		writeFatalError(w, http.StatusBadRequest,
			fmt.Sprintf("unable to decode request body; err: %s", err.Error()),
			"incorrect json")
		return
	}

	// нечего обновлять
	if updateForm.Username == "" && updateForm.NewPassword == "" {
		writeApplicationJSON(w, &apptypes.Errors{})
		return
	}

	user, errs := dblib.GetUser(map[string]interface{}{
		"id": vars["userID"],
	})
	if errs != nil {
		writeApplicationJSON(w, errs)
		return
	}

	if updateForm.Username != "" {
		user.Username = updateForm.Username
	}

	if updateForm.NewPassword != "" {
		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(updateForm.OldPassword)); err != nil {
			log.Warningf("user: %s wrong password", user.Username)
			writeApplicationJSON(w, &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"oldPassword": &apptypes.Error{
						Code:        apptypes.WrongPassword,
						Description: "Record Not Found",
					},
				},
			})
			return
		}

		newPass, err := bcrypt.GenerateFromPassword([]byte(updateForm.NewPassword), bcrypt.MinCost)
		if err != nil {
			writeFatalError(w, http.StatusInternalServerError,
				fmt.Sprintf("bcrypt hash error: %s", err.Error()),
				"internal server error")
			return
		}

		user.Password = newPass
	}

	if errs = user.Save(); errs != nil {
		writeApplicationJSON(w, errs)
		return
	}

	log.Noticef("user %d updated;", info.ID)
	writeApplicationJSON(w, &apptypes.Errors{})
}

// SignInUser signs in and returns the authentication cookie
func (h *Handler) SignInUser(w http.ResponseWriter, r *http.Request) {
	form := &apptypes.FormUser{}
	err := decodeBodyJSON(r.Body, form)
	if err != nil {
		writeFatalError(w, http.StatusBadRequest,
			fmt.Sprintf("unable to decode request body; err: %s", err.Error()),
			"incorrect json")
		return
	}

	user, errs := dblib.GetUser(map[string]interface{}{
		"username": form.Username,
	})
	if errs != nil {
		writeApplicationJSON(w, errs)
		return
	}

	if !user.Active {
		log.Warning("user was deleted recently")
		writeApplicationJSON(w, &apptypes.Errors{
			Other: &apptypes.Error{
				Code:        apptypes.NotActive,
				Description: "User is not active",
			},
		})
		return
	}

	// проверяем пароли
	if err := bcrypt.CompareHashAndPassword(user.Password,
		[]byte(form.Password)); err != nil {
		log.Warningf("user: %s wrong password", user.Username)
		writeApplicationJSON(w, &apptypes.Errors{
			Fields: map[string]*apptypes.Error{
				"password": &apptypes.Error{
					Code:        apptypes.WrongPassword,
					Description: "Record Not Found",
				},
			},
		})
		return
	}

	session := dblib.Session{
		Info: &apptypes.InfoUser{
			ID:     user.ID,
			Active: user.Active,
			BasicUser: apptypes.BasicUser{
				Username: user.Username,
			},
		},
		ExpiresAfter: time.Hour * 24 * 30,
	}
	errs = session.Set()
	if errs != nil {
		writeApplicationJSON(w, errs)
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

	log.Noticef("username %s signin ok", user.Username)
}

// SignOutUser signs out and deletes the authentication cookie
func (h *Handler) SignOutUser(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("JSESSIONID")
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("cant get cookie; err: %s", err.Error()),
			"internal server error")
		return
	}

	session := dblib.Session{
		Token: cookie.Value,
	}
	errs := session.Delete()
	if errs != nil {
		writeApplicationJSON(w, errs)
		return
	}

	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)

	log.Noticef("token %s removed", cookie.Value)
	writeApplicationJSON(w, &apptypes.Errors{})
}

// SignUpUser creates new user
func (h *Handler) SignUpUser(w http.ResponseWriter, r *http.Request) {
	form := &apptypes.FormUser{}
	err := decodeBodyJSON(r.Body, form)
	if err != nil {
		writeFatalError(w, http.StatusBadRequest,
			fmt.Sprintf("unable to decode request body; err: %s", err.Error()),
			"incorrect json")
		return
	}

	user := dblib.User{
		Username: form.Username,
	}

	user.Password, err = bcrypt.GenerateFromPassword([]byte(form.Password),
		bcrypt.MinCost)
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("bcrypt hash error: %s", err.Error()),
			"internal server error")
		return
	}

	errs := user.Create()
	if errs != nil {
		writeApplicationJSON(w, errs)
		return
	}

	log.Noticef("user %s created", user.Username)
	writeApplicationJSON(w, &apptypes.Errors{})
}

func main() {
	//setting logs format
	backendLog := logging.NewLogBackend(os.Stderr, "", 0)
	logging.SetBackend(logging.NewBackendFormatter(backendLog, logFormat))

	dblib.ConnectDB("warscript_user", "qwerty", "localhost", "warscript_db")
	dblib.ConnectStorage("user", "", "localhost", 6379)

	//setting templates
	h := &Handler{
		Tmpls:            make(map[string]*template.Template),
		DBConn:           dblib.GetDB(),
		SessionStoreConn: dblib.GetStorage(),
	}
	h.Tmpls["index.html"] = template.Must(template.ParseFiles("templates/tmp.html"))

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("static/"))))
	r.HandleFunc("/", h.Index).Methods("GET")
	r.HandleFunc("/signup", h.SignUpUser).Methods("POST")
	r.HandleFunc("/signin", h.SignInUser).Methods("POST")
	r.HandleFunc("/signout", WithAuthentication(h.SignOutUser, h)).Methods("POST")
	r.HandleFunc("/users/username_check", h.CheckUsername).Methods("POST")
	r.HandleFunc("/users/{userID:[0-9]+}", h.GetUser).Methods("GET")
	r.HandleFunc("/users/{userID:[0-9]+}", WithAuthentication(h.UpdateUser, h)).Methods("POST")
	//r.HandleFunc("/users/{userID:[0-9]+}/delete", //temproraty deprecated
	//	WithAuthentication(h.DeleteUser, h)).Methods("POST")

	h.Router = AccessLogMiddleware(r)
	log.Noticef("MainService successfully started at port %d", os.Getenv("MAIN_PORT"))
	err := http.ListenAndServe(os.Getenv("MAIN_PORT"), h.Router)
	if err != nil {
		log.Criticalf("cant start main server. err: %s", err.Error())
		return
	}
}
