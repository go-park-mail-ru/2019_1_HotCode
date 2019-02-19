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
	err = writeApplicationJSON(w, &struct {
		Used bool `json:"used"`
	}{
		Used: used,
	})
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("result marshal error: %s", err.Error()),
			"internal server error")
		return
	}

	log.Noticef("username %s check ok", username.Username)
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

	errors := &FromErrors{
		Errors: make(map[string]*Error),
	}

	if user.Validate(errors.Errors) {
		storedUser := &User{}
		// ищем юзера с таким именем
		if dbc := h.DBConn.First(storedUser, "username = ?", user.Username); dbc.RecordNotFound() {
			// не нашли
			log.Errorf("user not found: %s", dbc.Error.Error())
			errors.Errors["other"] = &Error{
				Code:        3,
				Message:     "Wrong username or password",
				Description: "Record Not Found",
			}
			writeFormErrorsJSON(w, &errors)
			return
		} else if dbc.Error != nil {
			// ошибка в базе
			writeFatalError(w, http.StatusInternalServerError,
				fmt.Sprintf("database first error: %s", dbc.Error.Error()),
				"internal server error")
			return
		}

		// проверяем пароли
		if err := bcrypt.CompareHashAndPassword(storedUser.PasswordEncoded, []byte(user.PasswordRaw)); err != nil {
			log.Errorf("user: %s wrong password", user.Username)
			errors.Errors["other"] = &Error{
				Code:        3,
				Message:     "Wrong username or password",
				Description: "Record Not Found",
			}
			writeFormErrorsJSON(w, &errors)
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

		// на 30 суток
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
	}

	writeFormErrorsJSON(w, &errors)
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

	errors := &FromErrors{
		Errors: make(map[string]*Error),
	}

	if user.Validate(errors.Errors) {
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
				errors.Errors["username"] = &Error{
					Code:        1,
					Message:     "Username already used!",
					Description: errorStr,
				}
				writeFormErrorsJSON(w, &errors)
				return
			}
		}

		log.Noticef("user %s created", user.Username)
	}

	writeFormErrorsJSON(w, &errors)
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
	r.HandleFunc("/users/username_check", h.CheckUsername).Methods("POST")

	h.Router = AccessLogMiddleware(r)
	log.Noticef("MainService successfully started at port %d", configuration.Port)
	err = http.ListenAndServe(":"+strconv.Itoa(configuration.Port), h.Router)
	if err != nil {
		log.Criticalf("cant start main server. err: %s", err.Error())
		return
	}
}
