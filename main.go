package main

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"

	"golang.org/x/time/rate"

	"github.com/gorilla/mux"
	"github.com/jcftang/logentriesrus"

	"github.com/go-park-mail-ru/2019_1_HotCode/controllers"
	"github.com/go-park-mail-ru/2019_1_HotCode/models"

	log "github.com/sirupsen/logrus"
)

// docker run -d -p 6379:6379 redis
// docker kill $(docker ps -q)
// docker rm $(docker ps -a -q)

// Handler пока что только хранит темплейты
// потом можно добавить grpc клиенты
type Handler struct {
	Router http.Handler
}

// NewHandler creates new handler
func NewHandler() *Handler {
	h := &Handler{}

	// этот роутер будет отвечать за первую(и пока единственную) версию апишки
	r := mux.NewRouter().PathPrefix("/v1").Subrouter()

	r.HandleFunc("/sessions", controllers.WithAuthentication(controllers.GetSession)).Methods("GET")
	r.HandleFunc("/sessions", controllers.CreateSession).Methods("POST")
	r.HandleFunc("/sessions", controllers.WithAuthentication(controllers.DeleteSession)).Methods("DELETE")

	r.HandleFunc("/users", controllers.CreateUser).Methods("POST")
	r.HandleFunc("/users", controllers.WithAuthentication(controllers.UpdateUser)).Methods("PUT")
	r.HandleFunc("/users/{user_id:[0-9]+}", controllers.GetUser).Methods("GET")
	r.HandleFunc("/users/used", WithLimiter(controllers.CheckUsername, rate.NewLimiter(3, 5))).Methods("POST")

	r.HandleFunc("/games", controllers.GetGameList).Methods("GET")
	r.HandleFunc("/games/{game_id:[0-9]+}", controllers.GetGame).Methods("GET")
	r.HandleFunc("/games/{game_id:[0-9]+}/leaderboard", controllers.GetGameLeaderboard).Methods("GET")
	r.HandleFunc("/games/{game_id:[0-9]+}/leaderboard/count", controllers.GetGameTotalPlayers).Methods("GET")

	h.Router = RecoverMiddleware(AccessLogMiddleware(r))
	return h
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	// собираем логи в хранилище
	le, err := logentriesrus.NewLogentriesrusHook(os.Getenv("LOGENTRIESRUS_TOKEN"))
	if err != nil {
		os.Exit(-1)
	}
	log.AddHook(le)
}

func main() {
	err := models.ConnectDB(os.Getenv("DB_USER"), os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	if err != nil {
		log.Fatalf("failed to connect to db; err: %s", err.Error())
	}
	err = models.ConnectStorage(os.Getenv("STORAGE_USER"), os.Getenv("STORAGE_PASS"),
		os.Getenv("STORAGE_HOST"))
	if err != nil {
		log.Fatalf("cant connect to redis session storage; err: %s", err.Error())
	}

	h := NewHandler()

	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{os.Getenv("CORS_HOST")}),
		handlers.AllowedMethods([]string{"POST", "GET", "PUT", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
		handlers.AllowCredentials(),
	)

	port := os.Getenv("PORT")
	log.Printf("MainService successfully started at port %s", port)
	err = http.ListenAndServe(":"+port, corsMiddleware(h.Router))
	if err != nil {
		log.Fatalf("cant start main server. err: %s", err.Error())
		return
	}
}
