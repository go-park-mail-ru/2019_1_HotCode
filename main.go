package main

import (
	"net/http"
	"os"

	"github.com/go-park-mail-ru/2019_1_HotCode/queue"

	"github.com/gorilla/handlers"

	"golang.org/x/time/rate"

	"github.com/gorilla/mux"
	"github.com/jcftang/logentriesrus"

	"github.com/go-park-mail-ru/2019_1_HotCode/bots"
	"github.com/go-park-mail-ru/2019_1_HotCode/database"
	"github.com/go-park-mail-ru/2019_1_HotCode/games"
	"github.com/go-park-mail-ru/2019_1_HotCode/storage"
	"github.com/go-park-mail-ru/2019_1_HotCode/users"

	log "github.com/sirupsen/logrus"
)

// docker run -d -p 6379:6379 redis
// docker kill $(docker ps -q)
// docker rm $(docker ps -a -q)

// Handler dependency injection для роутера
type Handler struct {
	Router http.Handler
}

// NewHandler creates new handler
func NewHandler() *Handler {
	h := &Handler{}

	// этот роутер будет отвечать за первую(и пока единственную) версию апишки
	r := mux.NewRouter().PathPrefix("/v1").Subrouter()

	r.HandleFunc("/sessions", users.WithAuthentication(users.GetSession)).Methods("GET")
	r.HandleFunc("/sessions", users.CreateSession).Methods("POST")
	r.HandleFunc("/sessions", users.WithAuthentication(users.DeleteSession)).Methods("DELETE")

	r.HandleFunc("/users", users.CreateUser).Methods("POST")
	r.HandleFunc("/users", users.WithAuthentication(users.UpdateUser)).Methods("PUT")
	r.HandleFunc("/users/{user_id:[0-9]+}", users.GetUser).Methods("GET")
	r.HandleFunc("/users/used", WithLimiter(users.CheckUsername, rate.NewLimiter(3, 5))).Methods("POST")

	r.HandleFunc("/games", games.GetGameList).Methods("GET")
	r.HandleFunc("/games/{game_slug}", games.GetGame).Methods("GET")
	r.HandleFunc("/games/{game_slug}/leaderboard", games.GetGameLeaderboard).Methods("GET")
	r.HandleFunc("/games/{game_slug}/leaderboard/count", games.GetGameTotalPlayers).Methods("GET")

	r.HandleFunc("/bots", users.WithAuthentication(bots.CreateBot)).Methods("POST")
	r.HandleFunc("/bots", users.WithAuthentication(bots.GetBotsList)).Methods("GET")
	r.HandleFunc("/bots/verification", users.WithAuthentication(bots.OpenVerifyWS)).Methods("GET")

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
	err := database.Connect(os.Getenv("DB_USER"), os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	if err != nil {
		log.Fatalf("failed to connect to db; err: %s", err.Error())
	}
	defer database.Close()

	err = storage.Connect(os.Getenv("STORAGE_USER"), os.Getenv("STORAGE_PASS"),
		os.Getenv("STORAGE_HOST"))
	if err != nil {
		log.Fatalf("cant connect to session storage; err: %s", err.Error())
	}
	defer storage.Close()

	err = queue.Connect(os.Getenv("QUEUE_USER"), os.Getenv("QUEUE_PASS"),
		os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT"))
	if err != nil {
		log.Fatalf("can not connect to queue processor: %s", err.Error())
	}
	defer queue.Close()

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
