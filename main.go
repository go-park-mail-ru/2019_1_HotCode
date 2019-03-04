package main

import (
	"net/http"
	"os"

	"golang.org/x/time/rate"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"

	"2019_1_HotCode/controllers"
	"2019_1_HotCode/models"

	log "github.com/sirupsen/logrus"
)

// docker run -d -p 6379:6379 redis
// docker kill $(docker ps -q)
// docker rm $(docker ps -a -q)

// Handler пока что только хранит темплейты
// потом можно добавить grpc клиенты
type Handler struct {
	Router           http.Handler
	DBConn           *gorm.DB
	SessionStoreConn redis.Conn
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
}

func main() {
	err := models.ConnectDB("warscript_user", "qwerty", "localhost", "warscript_db")
	if err != nil {
		log.Fatalf("failed to connect to db; err: %s", err.Error())
	}
	err = models.ConnectStorage("user", "", "localhost:6379")
	if err != nil {
		log.Fatalf("cant connect to redis session storage; err: %s", err.Error())
	}

	//setting templates
	h := &Handler{
		DBConn:           models.GetDB(),
		SessionStoreConn: models.GetStorage(),
	}

	// этот роутер будет отвечать за первую(и пока единственную) версию апишки
	r := mux.NewRouter().PathPrefix("/v1").Subrouter()

	r.HandleFunc("/sessions", WithAuthentication(controllers.GetSession)).Methods("GET")
	r.HandleFunc("/sessions", controllers.SignInUser).Methods("POST")
	r.HandleFunc("/sessions", WithAuthentication(controllers.SignOutUser)).Methods("DELETE")

	r.HandleFunc("/users", controllers.CreateUser).Methods("POST")
	r.HandleFunc("/users", WithAuthentication(controllers.UpdateUser)).Methods("PUT")
	r.HandleFunc("/users/{user_id:[0-9]+}", controllers.GetUser).Methods("GET")
	r.HandleFunc("/users/used", WithLimiter(controllers.CheckUsername, rate.NewLimiter(1, 1))).Methods("POST")
	h.Router = AccessLogMiddleware(r)

	port := os.Getenv("PORT")
	log.Printf("MainService successfully started at port %s", port)
	err = http.ListenAndServe(":"+port, h.Router)
	if err != nil {
		log.Fatalf("cant start main server. err: %s", err.Error())
		return
	}
}
