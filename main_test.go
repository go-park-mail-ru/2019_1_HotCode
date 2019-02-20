package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/garyburd/redigo/redis"
	"github.com/jinzhu/gorm"
)

const (
	// DSN настройки соединения
	psqlTestStr = "postgres://warscript_test_user:qwerty@localhost/warscript_test_db"
	// тестовый на 6380
	redisTestStr = "redis://user:@localhost:6380/0"
)

const (
	reloadTableSQL = `
DROP TABLE IF EXISTS "user";
create table "user"
(
	id bigserial not null
		constraint user_pk
			primary key,
	username varchar(32) CONSTRAINT username_empty not null check ( username <> '' ),
	password TEXT CONSTRAINT username_empty not null check ( password <> '' ),
	active boolean default true not null,
  CONSTRAINT uniq_username UNIQUE(username)
);

create unique index user_username_uindex
	on "user" (username);
`
)

type Case struct {
	Payload      []byte
	ExpectedCode int
	ExpectedBody string
	Method       string
	Endpoint     string
	Params       []interface{}
	Handler      http.HandlerFunc
}

func initHandler() Handler {
	//setting db connection
	//TODO: move it to lib
	db, _ := gorm.Open("postgres", psqlTestStr)
	db.LogMode(false)

	// Drop user table
	db.Exec(reloadTableSQL)

	sessionsRedisConn, _ := redis.DialURL(redisTestStr)

	sessionsRedisConn.Do("FLUSHDB")

	return Handler{
		DBConn:           db,
		SessionStoreConn: sessionsRedisConn,
	}
}

func makeRequest(method, endpoint string, cookies []*http.Cookie,
	body io.Reader, handle http.HandlerFunc) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, endpoint, body)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp := httptest.NewRecorder()
	handle(resp, req)
	return resp
}

func runTableAPITests(t *testing.T, cases []*Case) {
	for i, c := range cases {
		resp := makeRequest(c.Method, fmt.Sprintf(c.Endpoint, c.Params...),
			nil, bytes.NewBuffer(c.Payload), c.Handler)

		if resp.Code != c.ExpectedCode {
			t.Fatalf("\n[%d] Expected response code %d Got %d\n", i, c.ExpectedCode, resp.Code)
		}

		if resp.Body.String() != c.ExpectedBody {
			t.Fatalf("\n[%d] Expected response:\n %s\n Got:\n %s\n", i, c.ExpectedBody, resp.Body.String())
		}
	}
}

func TestSignUpUser(t *testing.T) {
	h := initHandler()

	cases := []*Case{
		{ //Всё ок
			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
			ExpectedCode: 200,
			ExpectedBody: `{}`,
			Method:       "POST",
			Endpoint:     "/signup",
			Handler:      h.SignUpUser,
		},
		{ // На используемый username
			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"errors":{"username":{"code":1,"message":"Username already used!","description":"pq: duplicate key value violates unique constraint \"uniq_username\""}}}`,
			Method:       "POST",
			Endpoint:     "/signup",
			Handler:      h.SignUpUser,
		},
		{ // Пустой юзернейм
			Payload:      []byte(`{"username":"","password":"dsadasd"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"errors":{"username":{"code":2,"message":"Username is empty","description":""}}}`,
			Method:       "POST",
			Endpoint:     "/signup",
			Handler:      h.SignUpUser,
		},
		{ // Пустой пароль
			Payload:      []byte(`{"username":"kek","password":""}`),
			ExpectedCode: 200,
			ExpectedBody: `{"errors":{"password":{"code":2,"message":"Password is empty","description":""}}}`,
			Method:       "POST",
			Endpoint:     "/signup",
			Handler:      h.SignUpUser,
		},
		{ // Неправильный формат JSON
			Payload:      []byte(`{"username":"kek""}`),
			ExpectedCode: 400,
			ExpectedBody: "incorrect json\n",
			Method:       "POST",
			Endpoint:     "/signup",
			Handler:      h.SignUpUser,
		},
	}

	runTableAPITests(t, cases)
}

func TestCheckUsername(t *testing.T) {
	h := initHandler()

	cases := []*Case{
		{ //Всё ок
			Payload:      []byte(`{"username":"sdas"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"used":false}`,
			Method:       "POST",
			Endpoint:     "/users/username_check",
			Handler:      h.CheckUsername,
		},
		{ // Создадим юзера
			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
			ExpectedCode: 200,
			ExpectedBody: `{}`,
			Method:       "POST",
			Endpoint:     "/signup",
			Handler:      h.SignUpUser,
		},
		{ // Теперь уже имя занято
			Payload:      []byte(`{"username":"sdas"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"used":true}`,
			Method:       "POST",
			Endpoint:     "/users/username_check",
			Handler:      h.CheckUsername,
		},
		{ // Пустой никнейм, очевидно, свободен, но зарегать его всё равно нельзя
			Payload:      []byte(`{"username":""}`),
			ExpectedCode: 200,
			ExpectedBody: `{"used":false}`,
			Method:       "POST",
			Endpoint:     "/users/username_check",
			Handler:      h.CheckUsername,
		},
		{ // Неправильный формат JSON
			Payload:      []byte(`{"username":"kek""}`),
			ExpectedCode: 400,
			ExpectedBody: "incorrect json\n",
			Method:       "POST",
			Endpoint:     "/users/username_check",
			Handler:      h.CheckUsername,
		},
	}

	runTableAPITests(t, cases)
}

func TestSignInUser(t *testing.T) {
	h := initHandler()

	cases := []*Case{
		{ //Такого юзера пока нет
			Payload:      []byte(`{"username":"kek","password":"lol"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"other":[{"code":3,"message":"Wrong username or password","description":"Record Not Found"}]}`,
			Method:       "POST",
			Endpoint:     "/signin",
			Handler:      h.SignInUser,
		},
		{ // Создадим юзера
			Payload:      []byte(`{"username":"kek","password":"lol"}`),
			ExpectedCode: 200,
			ExpectedBody: `{}`,
			Method:       "POST",
			Endpoint:     "/signup",
			Handler:      h.SignUpUser,
		},
		{ // Теперь юзер логинится
			Payload:      []byte(`{"username":"kek","password":"lol"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"username":"kek","id":1,"active":true}`,
			Method:       "POST",
			Endpoint:     "/signin",
			Handler:      h.SignInUser,
		},
		{ // Пустой никнейм нельзя
			Payload:      []byte(`{"username":"", "password":"lol"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"errors":{"username":{"code":2,"message":"Username is empty","description":""}}}`,
			Method:       "POST",
			Endpoint:     "/signin",
			Handler:      h.SignInUser,
		},
		{ // Пустой пароль тоже нельзя
			Payload:      []byte(`{"username":"kek", "password":""}`),
			ExpectedCode: 200,
			ExpectedBody: `{"errors":{"password":{"code":2,"message":"Password is empty","description":""}}}`,
			Method:       "POST",
			Endpoint:     "/signin",
			Handler:      h.SignInUser,
		},
		{ // Неправильный формат JSON
			Payload:      []byte(`{"username":"kek""}`),
			ExpectedCode: 400,
			ExpectedBody: "incorrect json\n",
			Method:       "POST",
			Endpoint:     "/signin",
			Handler:      h.SignInUser,
		},
	}

	runTableAPITests(t, cases)
}
