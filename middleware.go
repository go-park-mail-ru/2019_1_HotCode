package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2019_1_HotCode/controllers"
	"github.com/go-park-mail-ru/2019_1_HotCode/models"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// RecoverMiddleware ловит паники и кидает 500ки
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				utils.WriteApplicationJSON(w, http.StatusInternalServerError,
					controllers.NewAPIError(models.ErrInternal))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware добавляет настройки CORS в хедер
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://20191hotcode-6t88u924a.now.sh")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		next.ServeHTTP(w, r)
	})
}

// AccessLogMiddleware логирование всех запросов
func AccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _ := uuid.NewV4()
		ctx := context.WithValue(r.Context(), controllers.RequestUUIDKey, token.String())

		start := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))
		log.WithFields(log.Fields{
			"token":       token.String(),
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"work_time":   time.Since(start).Seconds(),
		}).Info(r.URL.Path)
	})
}

// WithAuthentication проверка токена перед исполнением запроса
//nolint: interfacer
func WithAuthentication(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("JSESSIONID")
		if cookie == nil {
			utils.WriteApplicationJSON(w, http.StatusUnauthorized,
				controllers.NewAPIError(controllers.ErrUnauthorized))
			return
		}

		session, err := models.GetSession(cookie.Value)
		if err != nil {
			utils.WriteApplicationJSON(w, http.StatusInternalServerError,
				controllers.NewAPIError(err))
			return

		}
		user := &controllers.InfoUser{}
		err = json.Unmarshal(session.Payload, user)
		if err != nil {
			utils.WriteApplicationJSON(w, http.StatusInternalServerError,
				controllers.NewAPIError(err))
			return

		}

		ctx := context.WithValue(r.Context(), controllers.UserInfoKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithLimiter для запросов, у которых есть ограничение в секунду
//nolint: interfacer
func WithLimiter(next http.HandlerFunc, limiter *rate.Limiter) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
