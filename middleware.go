package main

import (
	"context"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2019_1_HotCode/controllers"
	"github.com/go-park-mail-ru/2019_1_HotCode/models"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// RecoverMiddleware ловит паники и кидает 500ки
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.WithField("method", "RECOVER").Error(err)
				controllers.NewErrorResponseWriter(w, log.WithField("method", "RecoverMiddleware")).
					WriteError(http.StatusInternalServerError, models.ErrInternal)
			}
		}()

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
			"token":       token.String()[:8],
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
			"work_time":   time.Since(start).Seconds(),
		}).Info(r.URL.Path)
	})
}

// WithLimiter для запросов, у которых есть ограничение в секунду
//nolint: interfacer
func WithLimiter(next http.HandlerFunc, limiter *rate.Limiter) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			log.WithField("method", "WithLimiter").Warn("too many requests")
			http.Error(w, "", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
