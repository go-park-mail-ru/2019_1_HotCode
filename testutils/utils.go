package testutils

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

type Case struct {
	Payload      []byte
	ExpectedCode int
	ExpectedBody string
	Method       string
	Endpoint     string
	Pattern      string
	Cookies      []*http.Cookie
	Function     http.HandlerFunc
	Context      context.Context
}

func MakeRequest(ctx context.Context, handler http.Handler, method, endpoint string, cookies []*http.Cookie,
	body io.Reader) *httptest.ResponseRecorder {

	req, _ := http.NewRequest(method, endpoint, body)
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

func RunAPITest(t *testing.T, i int, c *Case) {
	r := mux.NewRouter()
	r.HandleFunc(c.Pattern, c.Function).Methods(c.Method)
	if c.Endpoint == "" {
		c.Endpoint = c.Pattern
	}
	resp := MakeRequest(c.Context, r, c.Method, c.Endpoint,
		c.Cookies, bytes.NewBuffer(c.Payload))
	if resp.Code != c.ExpectedCode {
		t.Fatalf("\n[%d] Expected response code %d Got %d\n\n[%d] Expected response:\n %s\n Got:\n %s\n",
			i, c.ExpectedCode, resp.Code, i, c.ExpectedBody, resp.Body.String())
	}

	if resp.Body.String() != c.ExpectedBody {
		t.Fatalf("\n[%d] Expected response:\n %s\n Got:\n %s\n", i, c.ExpectedBody, resp.Body.String())
	}
}

func RunTableAPITests(t *testing.T, cases []*Case) {
	for i, c := range cases {
		RunAPITest(t, i, c)
	}
}
