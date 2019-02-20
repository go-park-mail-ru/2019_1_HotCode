package main

import (
	"net/http"
	"os"

	"github.com/op/go-logging"
)

// AccessLogMiddleware логирование всех запросов
func AccessLogMiddleware(next http.Handler) http.Handler {
	//setting logs format
	backendLog := logging.NewLogBackend(os.Stderr, "", 0)
	logging.SetBackend(logging.NewBackendFormatter(backendLog, logFormat))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("[%s] %s %s", r.Method, r.RemoteAddr, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// WithAuthentication проверка токена перед исполнением запроса
func WithAuthentication(next http.HandlerFunc, h *Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("JSESSIONID")
		if err != nil {
			log.Warningf("[%s] %s %s; AUTH FAILED; No cookie",
				r.Method, r.RemoteAddr, r.URL.Path)
			http.Error(w, "wrong token", http.StatusUnauthorized)
			return
		}
		info, err := h.CheckToken(cookie.Value)
		if err != nil {
			log.Warningf("[%s] %s %s; AUTH FAILED; JSESSIONID: %s",
				r.Method, r.RemoteAddr, r.URL.Path, cookie.Value)
			http.Error(w, "wrong token", http.StatusUnauthorized)
			return
		}

		log.Noticef("username %s; session %s; check ok", info.Username, cookie.Value)
		next.ServeHTTP(w, r)
	})
}
