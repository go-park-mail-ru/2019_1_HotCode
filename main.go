package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	logging "github.com/op/go-logging"
	uuid "github.com/satori/go.uuid"
	"github.com/tkanos/gonfig"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

const (
	//DSN настройки соединения
	psqlStr = "postgres://warscript_user:qwerty@localhost/warscript_db"

	// docker run -d -p 6379:6379 redis
	// docker kill $$(docker ps -q)
	// docker rm $$(docker ps -a -q)
	redisDSN = "redis://user:@localhost:6379/0"
)

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
func (h *Handler) CheckToken(sessionToken string) (*InfoUser, error) {
	// запросы уедут в либу
	data, err := redis.Bytes(h.SessionStoreConn.Do("GET", sessionToken))
	if err != nil {
		return nil, err
	}

	userInfo := &InfoUser{}
	err = json.Unmarshal(data, userInfo)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

// CheckUsername checks if username already used
func (h *Handler) CheckUsername(w http.ResponseWriter, r *http.Request) {
	username := &BasicUser{}
	err := decodeBodyJSON(r.Body, username)
	if err != nil {
		writeFatalError(w, http.StatusBadRequest,
			fmt.Sprintf("unable to decode request body; err: %s", err.Error()),
			"incorrect json")
		return
	}

	used := !h.DBConn.First(&BasicUser{}, "username = ?", username.Username).RecordNotFound()
	writeApplicationJSON(w, &struct {
		Used bool `json:"used"`
	}{
		Used: used,
	})

	log.Noticef("username %s check ok; USED: %t", username.Username, used)
}

// GetUser get user info by ID
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	//вот это всё уложить в либу
	userInfo := &InfoUser{}
	if dbc := h.DBConn.First(userInfo, "id = ?", vars["userID"]); dbc.RecordNotFound() ||
		!userInfo.Active {
		log.Warningf("user %s not found", vars["userID"])
		writeApplicationJSON(w, &FromErrors{
			Errors: map[string]*Error{
				"userID": &Error{
					Code:        5,
					Message:     "User was not created or deleted",
					Description: "Record not found or active false",
				},
			},
		})
		return
	} else if dbc.Error != nil {
		// ошибка в базе
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("database first error: %s", dbc.Error.Error()),
			"internal server error")
		return
	}

	writeApplicationJSON(w, userInfo)
	log.Noticef("user %s was found", vars["userID"])
}

// UpdateUser updates user info by ID
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	info := userInfo(r)

	if vars["userID"] != strconv.Itoa(int(info.ID)) {
		writeFatalError(w, http.StatusForbidden,
			fmt.Sprintf("%s tried to change %d", vars["userID"], info.ID),
			"you don't have permission to this page")
		return
	}

	updateForm := &struct {
		BasicUser
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
		writeApplicationJSON(w, &FromErrors{})
		return
	}

	//вот это всё уложить в либу
	storedUser := &User{}
	if dbc := h.DBConn.First(storedUser, "id = ?", vars["userID"]); dbc.RecordNotFound() ||
		!storedUser.Active {
		log.Warningf("user %s not found", vars["userID"])
		writeApplicationJSON(w, &FromErrors{
			Errors: map[string]*Error{
				"userID": &Error{
					Code:        5,
					Message:     "User was not created or deleted",
					Description: "Record not found or active false",
				},
			},
		})
		return
	} else if dbc.Error != nil {
		// ошибка в базе
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("database first error: %s", dbc.Error.Error()),
			"internal server error")
		return
	}

	if updateForm.Username != "" {
		storedUser.Username = updateForm.Username
	}

	if updateForm.NewPassword != "" {
		if err := bcrypt.CompareHashAndPassword(storedUser.PasswordEncoded, []byte(updateForm.OldPassword)); err != nil {
			log.Warningf("user: %s wrong password", storedUser.Username)
			writeApplicationJSON(w, &FromErrors{
				Other: []*Error{
					&Error{
						Code:        3,
						Message:     "Wrong password",
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

		storedUser.PasswordEncoded = newPass
	}

	if dbc := h.DBConn.Save(storedUser); dbc.Error != nil {
		errorStr := dbc.Error.Error()
		log.Errorf("database create error: %s", errorStr)

		//TODO: спрятать это в либу
		if strings.Index(errorStr, "uniq_username") != -1 {
			writeApplicationJSON(w, &FromErrors{
				Errors: map[string]*Error{
					"username": &Error{
						Code:        1,
						Message:     "Username already used!",
						Description: errorStr,
					},
				},
			})
			return
		}

		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("database create error: %s", errorStr),
			"internal server error")
		return
	}

	log.Noticef("user %d updated;", info.ID)
	writeApplicationJSON(w, &FromErrors{})
}

// SignInUser signs in and returns the authentication cookie
func (h *Handler) SignInUser(w http.ResponseWriter, r *http.Request) {
	user := &FormUser{}
	err := decodeBodyJSON(r.Body, user)
	if err != nil {
		writeFatalError(w, http.StatusBadRequest,
			fmt.Sprintf("unable to decode request body; err: %s", err.Error()),
			"incorrect json")
		return
	}

	if errs, ok := user.Validate(); !ok {
		log.Warning("user form validation failed")
		writeApplicationJSON(w, errs)
		return
	}

	//тоже уйдёт в либу
	storedUser := &User{}
	// ищем юзера с таким именем
	if dbc := h.DBConn.First(storedUser, "username = ?", user.Username); dbc.RecordNotFound() {
		// не нашли или удалён
		log.Warning("user not found")
		writeApplicationJSON(w, &FromErrors{
			Other: []*Error{
				&Error{
					Code:        3,
					Message:     "Wrong username or password",
					Description: "Record Not Found",
				},
			},
		})
		return
	} else if dbc.Error != nil {
		// ошибка в базе
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("database first error: %s", dbc.Error.Error()),
			"internal server error")
		return
	}

	if !storedUser.Active {
		log.Warning("user was deleted recently")
		writeApplicationJSON(w, &FromErrors{
			Other: []*Error{
				&Error{
					Code:        4,
					Message:     "User was deleted recently",
					Description: "Active is false",
				},
			},
		})
		return
	}

	// проверяем пароли
	if err := bcrypt.CompareHashAndPassword(storedUser.PasswordEncoded, []byte(user.PasswordRaw)); err != nil {
		log.Warningf("user: %s wrong password", user.Username)
		writeApplicationJSON(w, &FromErrors{
			Other: []*Error{
				&Error{
					Code:        3,
					Message:     "Wrong username or password",
					Description: "Record Not Found",
				},
			},
		})
		return
	}

	// записываем токен и инфу
	sessionToken, err := uuid.NewV4()
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("session token generate error: %s", err.Error()),
			"internal server error")
		return
	}
	sessionInfo, err := json.Marshal(&InfoUser{
		ID:     storedUser.ID,
		Active: storedUser.Active,
		BasicUser: BasicUser{
			Username: storedUser.Username,
		},
	})
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("session info marshal error: %s", err.Error()),
			"internal server error")
		return
	}

	// на 30 суток(убрать в либу)
	_, err = h.SessionStoreConn.Do("SETEX", sessionToken.String(), "2628000", sessionInfo)
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("session store save error: %s", err.Error()),
			"internal server error")
		return
	}

	// ставим куку
	http.SetCookie(w, &http.Cookie{
		Name:    "JSESSIONID",
		Value:   sessionToken.String(),
		Expires: time.Now().Add(2628000 * time.Second),
	})

	//уже есть готовая последовательность байт
	w.Header().Set("Content-Type", "application/json")
	w.Write(sessionInfo)

	log.Noticef("username %s signin ok", storedUser.Username)
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

	//убрать в либу
	_, err = h.SessionStoreConn.Do("DEL", cookie.Value)
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("cant delete cookie; err: %s", err.Error()),
			"internal server error")
		return
	}

	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)

	log.Noticef("token %s removed", cookie.Value)
	writeApplicationJSON(w, &FromErrors{})
}

// SignUpUser creates new user
func (h *Handler) SignUpUser(w http.ResponseWriter, r *http.Request) {
	user := &FormUser{}
	err := decodeBodyJSON(r.Body, user)
	if err != nil {
		writeFatalError(w, http.StatusBadRequest,
			fmt.Sprintf("unable to decode request body; err: %s", err.Error()),
			"incorrect json")
		return
	}

	if errs, ok := user.Validate(); !ok {
		log.Warning("user form validation failed")
		writeApplicationJSON(w, errs)
		return
	}

	user.PasswordEncoded, err = bcrypt.GenerateFromPassword([]byte(user.PasswordRaw), bcrypt.MinCost)
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("bcrypt hash error: %s", err.Error()),
			"internal server error")
		return
	}

	if dbc := h.DBConn.Create(user); dbc.Error != nil {
		errorStr := dbc.Error.Error()
		log.Errorf("database create error: %s", errorStr)

		//TODO: спрятать это в либу
		if strings.Index(errorStr, "uniq_username") != -1 {
			writeApplicationJSON(w, &FromErrors{
				Errors: map[string]*Error{
					"username": &Error{
						Code:        1,
						Message:     "Username already used!",
						Description: errorStr,
					},
				},
			})
			return
		}

		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("database create error: %s", errorStr),
			"internal server error")
		return
	}

	log.Noticef("user %s created", user.Username)
	writeApplicationJSON(w, &FromErrors{})
}

func main() {
	//load server configuration
	configuration := Configuration{}
	err := gonfig.GetConf("config/config.development.json", &configuration)
	if err != nil {
		log.Fatalf("cant load config; err: %s", err.Error())
	}

	//setting logs format
	backendLog := logging.NewLogBackend(os.Stderr, "", 0)
	logging.SetBackend(logging.NewBackendFormatter(backendLog, logFormat))

	//setting db connection
	//TODO: move it to lib
	db, err := gorm.Open("postgres", psqlStr)
	if err != nil {
		log.Fatalf("cant open database connection; err: %s", err.Error())
	}
	db.LogMode(false)

	sessionsRedisConn, err := redis.DialURL(redisDSN)
	if err != nil {
		log.Fatalf("cant connect to redis session storage; err: %s", err.Error())
	}

	//setting templates
	h := &Handler{
		Tmpls:            make(map[string]*template.Template),
		DBConn:           db,
		SessionStoreConn: sessionsRedisConn,
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
	log.Noticef("MainService successfully started at port %d", configuration.Port)
	err = http.ListenAndServe(":"+strconv.Itoa(configuration.Port), h.Router)
	if err != nil {
		log.Criticalf("cant start main server. err: %s", err.Error())
		return
	}
}
