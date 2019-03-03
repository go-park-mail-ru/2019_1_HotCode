package main

import (
	"2019_1_HotCode/controllers"
	"2019_1_HotCode/models"
	"2019_1_HotCode/utils"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/time/rate"
)

// AccessLogMiddleware логирование всех запросов
func AccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.Method, r.RemoteAddr, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// WithAuthentication проверка токена перед исполнением запроса
func WithAuthentication(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("JSESSIONID")
		if cookie == nil {
			utils.WriteApplicationJSON(w, http.StatusUnauthorized,
				controllers.NewAPIError(controllers.Unauthorized))
			return
		}

		session, err := models.GetSession(cookie.Value)
		if err != nil {
			utils.WriteApplicationJSON(w, http.StatusInternalServerError,
				controllers.NewAPIError(err.Error()))
			return

		}
		user := &controllers.InfoUser{}
		err = json.Unmarshal(session.Payload, user)
		if err != nil {
			utils.WriteApplicationJSON(w, http.StatusInternalServerError,
				controllers.NewAPIError(err.Error()))
			return

		}

		ctx := context.WithValue(r.Context(), controllers.UserInfoKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithLimiter для запросов, у которых есть ограничение в секунду
func WithLimiter(next http.HandlerFunc, limiter *rate.Limiter) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter.Allow() == false {
			http.Error(w, "", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
