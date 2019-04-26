package users

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/pkg/errors"
)

// WithAuthentication проверка токена перед исполнением запроса
//nolint: interfacer
func WithAuthentication(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := utils.GetLogger(r, "CheckUsername")
		errWriter := utils.NewErrorResponseWriter(w, logger)

		cookie, err := r.Cookie("JSESSIONID")
		if err != nil || cookie == nil {
			errWriter.WriteWarn(http.StatusUnauthorized, errors.Wrap(err, "can not load cookie"))
			return
		}

		session, err := Sessions.GetSession(cookie.Value)
		if err != nil {
			errWriter.WriteError(http.StatusUnauthorized, errors.Wrap(err, "get session error"))
			return

		}
		payload := &SessionPayload{}
		err = json.Unmarshal(session.Payload, payload)
		if err != nil {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "session payload unmarshal error"))
			return

		}

		ctx := context.WithValue(r.Context(), SessionInfoKey, payload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
