package main

// import (
// 	"2019_1_HotCode/apptypes"
// 	"2019_1_HotCode/dblib"
// 	"bytes"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"

// 	"github.com/gorilla/mux"
// )

// const (
// 	// DSN настройки соединения
// 	psqlTestStr = "postgres://warscript_test_user:qwerty@localhost/warscript_test_db"
// 	// тестовый на 6380
// 	redisTestStr = "redis://user:@localhost:6380/0"
// )

// const (
// 	reloadTableSQL = `
// DROP TABLE IF EXISTS "user";
// create table "user"
// (
// 	id bigserial not null
// 		constraint user_pk
// 			primary key,
// 	username varchar(32) CONSTRAINT username_empty not null check ( username <> '' ),
// 	password TEXT CONSTRAINT username_empty not null check ( password <> '' ),
// 	active boolean default true not null,
//   CONSTRAINT uniq_username UNIQUE(username)
// );

// create unique index user_username_uindex
// 	on "user" (username);
// `
// )

// type Case struct {
// 	Payload      []byte
// 	ExpectedCode int
// 	ExpectedBody string
// 	Method       string
// 	Endpoint     string
// }

// func initHandler() *Handler {
// 	//setting db connection & Drop user table
// 	dblib.ConnectDB("warscript_test_user", "qwerty", "localhost", "warscript_test_db")
// 	dblib.GetDB().Exec(reloadTableSQL)

// 	dblib.ConnectStorage("user", "", "localhost", 6380)
// 	dblib.GetStorage().Do("FLUSHDB")

// 	h := &Handler{
// 		DBConn:           dblib.GetDB(),
// 		SessionStoreConn: dblib.GetStorage(),
// 	}

// 	r := mux.NewRouter()
// 	r.HandleFunc("/signup", h.SignUpUser).Methods("POST")
// 	r.HandleFunc("/signin", h.SignInUser).Methods("POST")
// 	r.HandleFunc("/signout", WithAuthentication(h.SignOutUser, h)).Methods("POST")
// 	r.HandleFunc("/users/username_check", h.CheckUsername).Methods("POST")
// 	r.HandleFunc("/users/{userID:[0-9]+}", h.GetUser).Methods("GET")
// 	r.HandleFunc("/users/{userID:[0-9]+}", WithAuthentication(h.UpdateUser, h)).Methods("POST")
// 	//r.HandleFunc("/users/{userID:[0-9]+}/delete", //temproraty deprecated
// 	//	WithAuthentication(h.DeleteUser, h)).Methods("POST")

// 	h.Router = AccessLogMiddleware(r)

// 	return h
// }

// func makeRequest(handler http.Handler, method, endpoint string, cookies []*http.Cookie,
// 	body io.Reader) *httptest.ResponseRecorder {
// 	req, _ := http.NewRequest(method, endpoint, body)
// 	for _, cookie := range cookies {
// 		req.AddCookie(cookie)
// 	}

// 	resp := httptest.NewRecorder()
// 	handler.ServeHTTP(resp, req)
// 	return resp
// }

// func runTableAPITests(t *testing.T, h http.Handler, cases []*Case) {
// 	for i, c := range cases {
// 		resp := makeRequest(h, c.Method, c.Endpoint,
// 			nil, bytes.NewBuffer(c.Payload))

// 		if resp.Code != c.ExpectedCode {
// 			t.Fatalf("\n[%d] Expected response code %d Got %d\n", i, c.ExpectedCode, resp.Code)
// 		}

// 		if resp.Body.String() != c.ExpectedBody {
// 			t.Fatalf("\n[%d] Expected response:\n %s\n Got:\n %s\n", i, c.ExpectedBody, resp.Body.String())
// 		}
// 	}
// }

// func TestSignUpUser(t *testing.T) {
// 	h := initHandler()

// 	cases := []*Case{
// 		{ //Всё ок
// 			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{}`,
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 		{ // На используемый username
// 			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: fmt.Sprintf(`{"fields":{"username":{"code":%d,"message":"","description":"pq: duplicate key value violates unique constraint \"uniq_username\""}}}`, apptypes.AlreadyUsed),
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 		{ // Пустой юзернейм
// 			Payload:      []byte(`{"username":"","password":"dsadasd"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: fmt.Sprintf(`{"fields":{"username":{"code":%d,"message":"","description":"Username is empty"}}}`, apptypes.FailedToValidate),
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 		{ // Пустой пароль нас не смущает
// 			Payload:      []byte(`{"username":"kek","password":""}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{}`,
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 		{ // Неправильный формат JSON
// 			Payload:      []byte(`{"username":"kek""}`),
// 			ExpectedCode: 400,
// 			ExpectedBody: "incorrect json\n",
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 	}

// 	runTableAPITests(t, h.Router, cases)
// }

// func TestSignInUser(t *testing.T) {
// 	h := initHandler()

// 	cases := []*Case{
// 		{ //Такого юзера пока нет
// 			Payload:      []byte(`{"username":"kek","password":"lol"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: fmt.Sprintf(`{"other":{"code":%d,"message":"","description":"Can't find user with given parameters"}}`, apptypes.RowNotFound),
// 			Method:       "POST",
// 			Endpoint:     "/signin",
// 		},
// 		{ // Создадим юзера
// 			Payload:      []byte(`{"username":"kek","password":"lol"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{}`,
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 		{ // Теперь юзер логинится
// 			Payload:      []byte(`{"username":"kek","password":"lol"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{"username":"kek","id":1,"active":true}`,
// 			Method:       "POST",
// 			Endpoint:     "/signin",
// 		},
// 		{ // Пустой никнейм нельзя
// 			Payload:      []byte(`{"username":"", "password":"lol"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: fmt.Sprintf(`{"other":{"code":%d,"message":"","description":"Can't find user with given parameters"}}`, apptypes.RowNotFound),
// 			Method:       "POST",
// 			Endpoint:     "/signin",
// 		},
// 		{ // Неправильный формат JSON
// 			Payload:      []byte(`{"username":"kek""}`),
// 			ExpectedCode: 400,
// 			ExpectedBody: "incorrect json\n",
// 			Method:       "POST",
// 			Endpoint:     "/signin",
// 		},
// 	}

// 	runTableAPITests(t, h.Router, cases)
// }

// func TestSignOutUser(t *testing.T) {
// 	h := initHandler()
// 	// выходим без токена(ничего не получится)
// 	resp := makeRequest(h.Router, "POST", "/signout", nil, nil)
// 	if resp.Code != http.StatusUnauthorized {
// 		t.Fatalf("\n[0] Expected response code %d Got %d\n",
// 			http.StatusUnauthorized, resp.Code)
// 	}

// 	expected0 := "wrong token\n"
// 	if resp.Body.String() != expected0 {
// 		t.Fatalf("\n[0] Expected response:\n %s\n Got:\n %s\n", expected0, resp.Body.String())
// 	}

// 	// зарегали
// 	makeRequest(h.Router, "POST", "/signup",
// 		nil, bytes.NewBuffer([]byte(`{"username":"kek","password":"lol"}`)))

// 	// залогинились
// 	resp = makeRequest(h.Router, "POST", "/signin",
// 		nil, bytes.NewBuffer([]byte(`{"username":"kek","password":"lol"}`)))

// 	// разлогинились
// 	resp = makeRequest(h.Router, "POST", "/signout",
// 		resp.Result().Cookies(), nil)
// 	if resp.Code != http.StatusOK {
// 		t.Fatalf("\n[1] Expected response code %d Got %d\n",
// 			http.StatusOK, resp.Code)
// 	}

// 	expected1 := "{}"
// 	if resp.Body.String() != expected1 {
// 		t.Fatalf("\n[1] Expected response:\n %s\n Got:\n %s\n", expected1, resp.Body.String())
// 	}
// }

// func TestCheckUsername(t *testing.T) {
// 	h := initHandler()

// 	cases := []*Case{
// 		{ //Всё ок
// 			Payload:      []byte(`{"username":"sdas"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{"used":false}`,
// 			Method:       "POST",
// 			Endpoint:     "/users/username_check",
// 		},
// 		{ // Создадим юзера
// 			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{}`,
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 		{ // Теперь уже имя занято
// 			Payload:      []byte(`{"username":"sdas"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{"used":true}`,
// 			Method:       "POST",
// 			Endpoint:     "/users/username_check",
// 		},
// 		{ // Пустой никнейм, очевидно, свободен, но зарегать его всё равно нельзя
// 			Payload:      []byte(`{"username":""}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{"used":false}`,
// 			Method:       "POST",
// 			Endpoint:     "/users/username_check",
// 		},
// 		{ // Неправильный формат JSON
// 			Payload:      []byte(`{"username":"kek""}`),
// 			ExpectedCode: 400,
// 			ExpectedBody: "incorrect json\n",
// 			Method:       "POST",
// 			Endpoint:     "/users/username_check",
// 		},
// 	}

// 	runTableAPITests(t, h.Router, cases)
// }

// func TestGetUser(t *testing.T) {
// 	h := initHandler()
// 	cases := []*Case{
// 		{ // Такого юзера пока нет
// 			ExpectedCode: 200,
// 			ExpectedBody: fmt.Sprintf(`{"other":{"code":%d,"message":"","description":"Can't find user with given parameters"}}`,
// 				apptypes.RowNotFound),
// 			Method:   "GET",
// 			Endpoint: "/users/1",
// 		},
// 		{ // Создадим юзера
// 			Payload:      []byte(`{"username":"golang","password":"4ever"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{}`,
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 		{ //Всё ок
// 			ExpectedCode: 200,
// 			ExpectedBody: `{"username":"golang","id":1,"active":true}`,
// 			Method:       "GET",
// 			Endpoint:     "/users/1",
// 		},
// 	}

// 	runTableAPITests(t, h.Router, cases)
// }

// func TestUpdateUser(t *testing.T) {
// 	h := initHandler()
// 	cases := []*Case{
// 		{ // Нужно залогиниться, чтобы обновлять юзеров
// 			ExpectedCode: http.StatusUnauthorized,
// 			ExpectedBody: "wrong token\n",
// 			Method:       "POST",
// 			Endpoint:     "/users/1",
// 		},
// 		{ // Создадим первого юзера
// 			Payload:      []byte(`{"username":"golang","password":"4ever"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{}`,
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 		{ // Создадим второго юзера
// 			Payload:      []byte(`{"username":"scheme","password":"4ever"}`),
// 			ExpectedCode: 200,
// 			ExpectedBody: `{}`,
// 			Method:       "POST",
// 			Endpoint:     "/signup",
// 		},
// 	}
// 	runTableAPITests(t, h.Router, cases)

// 	// логинимся на первом юзере
// 	resp := makeRequest(h.Router, "POST", "/signin",
// 		nil, bytes.NewBuffer([]byte(`{"username":"golang","password":"4ever"}`)))
// 	cookies := resp.Result().Cookies()
// 	// обновляем username
// 	resp = makeRequest(h.Router, "POST", "/users/1",
// 		cookies, bytes.NewBuffer([]byte(`{"username":"Me"}`)))
// 	if resp.Code != http.StatusOK {
// 		t.Fatalf("\n[3] Expected response code %d Got %d\n",
// 			http.StatusOK, resp.Code)
// 	}
// 	if resp.Body.String() != "{}" {
// 		t.Fatalf("\n[3] Expected response:\n %s\n Got:\n %s\n", "{}", resp.Body.String())
// 	}

// 	cases1 := []*Case{
// 		{ // имя обновилось
// 			ExpectedCode: 200,
// 			ExpectedBody: `{"username":"Me","id":1,"active":true}`,
// 			Method:       "GET",
// 			Endpoint:     "/users/1",
// 		},
// 	}
// 	runTableAPITests(t, h.Router, cases1)

// 	// пытаемся сменить имя чужому акку
// 	resp = makeRequest(h.Router, "POST", "/users/2",
// 		cookies, bytes.NewBuffer([]byte(`{"username":"Zatrolen"}`)))
// 	if resp.Code != http.StatusForbidden {
// 		t.Fatalf("\n[4] Expected response code %d Got %d\n",
// 			http.StatusOK, resp.Code)
// 	}
// 	if resp.Body.String() != "you don't have permission to this page\n" {
// 		t.Fatalf("\n[4] Expected response:\n %s\n Got:\n %s\n",
// 			"you don't have permission to this page\n", resp.Body.String())
// 	}

// 	// пытаемся изменить пароль
// 	resp = makeRequest(h.Router, "POST", "/users/1",
// 		cookies, bytes.NewBuffer([]byte(`{"newPassword":"4ever and ever"}`)))
// 	if resp.Code != http.StatusOK {
// 		t.Fatalf("\n[5] Expected response code %d Got %d\n",
// 			http.StatusOK, resp.Code)
// 	}

// 	expected0 := fmt.Sprintf(`{"fields":{"oldPassword":{"code":%d,"message":"","description":"Wrong old passwrod"}}}`,
// 		apptypes.WrongPassword)
// 	if resp.Body.String() != expected0 {
// 		t.Fatalf("\n[5] Expected response:\n %s\n Got:\n %s\n", expected0, resp.Body.String())
// 	}

// 	// пытаемся изменить пароль(используем старый)
// 	resp = makeRequest(h.Router, "POST", "/users/1",
// 		cookies, bytes.NewBuffer([]byte(`{"newPassword":"4ever and ever","oldPassword":"4ever"}`)))
// 	if resp.Code != http.StatusOK {
// 		t.Fatalf("\n[6] Expected response code %d Got %d\n",
// 			http.StatusOK, resp.Code)
// 	}
// 	if resp.Body.String() != "{}" {
// 		t.Fatalf("\n[6] Expected response:\n %s\n Got:\n %s\n", "{}", resp.Body.String())
// 	}

// 	//выходим
// 	makeRequest(h.Router, "POST", "/signout",
// 		cookies, nil)

// 	// зашли с новыми данными
// 	resp = makeRequest(h.Router, "POST", "/signin",
// 		nil, bytes.NewBuffer([]byte(`{"username":"Me","password":"4ever and ever"}`)))
// 	if resp.Code != http.StatusOK {
// 		t.Fatalf("\n[6] Expected response code %d Got %d\n",
// 			http.StatusOK, resp.Code)
// 	}
// 	if resp.Body.String() != `{"username":"Me","id":1,"active":true}` {
// 		t.Fatalf("\n[6] Expected response:\n %s\n Got:\n %s\n", `{"username":"Me","id":1,"active":true}`, resp.Body.String())
// 	}
// }
