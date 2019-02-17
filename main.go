package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	logging "github.com/op/go-logging"
	"github.com/tkanos/gonfig"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

const (
	//DSN настройки соединения
	psqlStr = "postgres://warscript_user:qwerty@localhost/warscript_db"
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
	Tmpls  map[string]*template.Template
	DBConn *gorm.DB
}

// Index рисует индекс
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.Tmpls["index.html"].ExecuteTemplate(w, "tmp", struct{}{})
}

// SignUpUser creates new user
func (h *Handler) SignUpUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	user := &User{}
	err := decoder.Decode(user)
	if err != nil {
		log.Errorf("unable to decode request body; err: %s", err.Error())
		http.Error(w, "incorrect json", http.StatusBadRequest)
		return
	}

	errors := &FromErrors{
		Errors: make(map[string]*Error),
	}

	if user.Validate(errors.Errors) {
		user.PasswordEncoded, err = bcrypt.GenerateFromPassword([]byte(user.PasswordRaw), bcrypt.MinCost)
		if err != nil {
			log.Errorf("bcrypt hash error: %s", err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
			}
		}

		if len(errors.Errors) == 0 {
			log.Noticef("user %s created", user.Username)
		}
	}

	respJSON, err := json.Marshal(&errors)
	if err != nil {
		log.Errorf("result marshal error: %s", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respJSON)
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

	//setting templates
	h := &Handler{
		Tmpls:  make(map[string]*template.Template),
		DBConn: db,
	}
	h.Tmpls["index.html"] = template.Must(template.ParseFiles("templates/tmp.html"))

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("static/"))))
	r.HandleFunc("/", h.Index).Methods("GET")
	r.HandleFunc("/signup", h.SignUpUser).Methods("POST")

	handler := AccessLogMiddleware(r)
	log.Noticef("MainService successfully started at port %d", configuration.Port)
	err = http.ListenAndServe(":"+strconv.Itoa(configuration.Port), handler)
	if err != nil {
		log.Criticalf("cant start main server. err: %s", err.Error())
		return
	}
}
