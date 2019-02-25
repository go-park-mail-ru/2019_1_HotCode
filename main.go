package main

import (
	"net/http"
	"os"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	logging "github.com/op/go-logging"

	"2019_1_HotCode/controllers"
	"2019_1_HotCode/models"
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
	DBConn           *gorm.DB
	SessionStoreConn redis.Conn
}

// CheckToken метод проверяющий токен, возвращает инфу о юзере(без пароля)
// позже выделится в отдельный запрос
func (h *Handler) CheckToken(sessionToken string) (*models.InfoUser, *models.Errors) {
	session, errs := models.GetSession(sessionToken)
	if errs != nil {
		return nil, errs
	}

	return session.Info, nil
}

func main() {
	//setting logs format
	backendLog := logging.NewLogBackend(os.Stderr, "", 0)
	logging.SetBackend(logging.NewBackendFormatter(backendLog, logFormat))

	err := models.ConnectDB("warscript_user", "qwerty", "localhost", "warscript_db")
	if err != nil {
		log.Fatalf("failed to connect to db; err: %s", err.Error())
	}
	err = models.ConnectStorage("user", "", "localhost", 6379)
	if err != nil {
		log.Fatalf("cant connect to redis session storage; err: %s", err.Error())
	}

	//setting templates
	h := &Handler{
		DBConn:           models.GetDB(),
		SessionStoreConn: models.GetStorage(),
	}

	r := mux.NewRouter()
	r.HandleFunc("/signup", controllers.SignUpUser).Methods("POST")
	r.HandleFunc("/signin", controllers.SignInUser).Methods("POST")
	r.HandleFunc("/signout", WithAuthentication(controllers.SignOutUser, h)).Methods("POST")
	r.HandleFunc("/users/username_check", controllers.CheckUsername).Methods("POST")
	r.HandleFunc("/users/{userID:[0-9]+}", controllers.GetUser).Methods("GET")
	r.HandleFunc("/users/{userID:[0-9]+}", WithAuthentication(controllers.UpdateUser, h)).Methods("POST")
	//r.HandleFunc("/users/{userID:[0-9]+}/delete", //temproraty deprecated
	//	WithAuthentication(h.DeleteUser, h)).Methods("POST")
	h.Router = AccessLogMiddleware(r)

	port := os.Getenv("PORT")
	log.Noticef("MainService successfully started at port %s", port)
	err = http.ListenAndServe(":"+port, h.Router)
	if err != nil {
		log.Criticalf("cant start main server. err: %s", err.Error())
		return
	}
}
