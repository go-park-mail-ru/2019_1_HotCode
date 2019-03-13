package controllers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2019_1_HotCode/models"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"

	"github.com/pkg/errors"
)

var nextFail error

type UsersTest struct {
	users    map[int64]models.User
	nextFail error
}

// setFailureUser fails next request
func setFailureUser(err error) {
	us := models.Users.(*UsersTest)
	us.nextFail = err
}

func checkFailureUser() error {
	us := models.Users.(*UsersTest)
	if us.nextFail != nil {
		defer func() {
			us.nextFail = nil
		}()
		return us.nextFail
	}
	return nil
}

var curID int64

func newID() int64 {
	curID++
	return curID
}

// Create создаёт запись в базе с новыми полями
func (us *UsersTest) Create(u *models.User) error {
	if err := checkFailureUser(); err != nil {
		return err
	}
	u.Active = true
	u.ID = newID()
	us.users[u.ID] = *u
	return nil
}

// Save сохраняет юзера в базу
func (us *UsersTest) Save(u *models.User) error {
	if err := checkFailureUser(); err != nil {
		return err
	}

	us.users[u.ID] = *u
	return nil
}

// CheckPassword проверяет пароль у юзера и сохранённый в модели
func (us *UsersTest) CheckPassword(u *models.User, password string) bool {
	return *u.Password == password
}

// GetUserByID получает юзера по id
func (us *UsersTest) GetUserByID(id int64) (*models.User, error) {
	if err := checkFailureUser(); err != nil {
		return nil, err
	}
	var u models.User
	var ok bool
	if u, ok = us.users[id]; !ok {
		return nil, models.ErrNotExists
	}

	return &u, nil
}

// GetUserByUsername получает юзера по имени
func (us *UsersTest) GetUserByUsername(username string) (*models.User, error) {
	if err := checkFailureUser(); err != nil {
		return nil, err
	}
	var u models.User
	var ok bool

	for _, user := range us.users {
		if user.Username == username {
			ok = true
			u = user
		}
	}
	if !ok {
		return nil, models.ErrNotExists
	}

	return &u, nil
}

type SessionsTest struct {
	sessions map[string][]byte
	nextFail error
}

func setFailureSession(err error) {
	ss := models.Sessions.(*SessionsTest)
	ss.nextFail = err
}

func checkFailureSession() error {
	ss := models.Sessions.(*SessionsTest)
	if ss.nextFail != nil {
		defer func() {
			ss.nextFail = nil
		}()
		return ss.nextFail
	}
	return nil
}

// Set валидирует и сохраняет сессию в хранилище по сгенерированному токену
// Токен сохраняется в s.Token
func (ss *SessionsTest) Set(s *models.Session) error {
	if err := checkFailureSession(); err != nil {
		return err
	}

	sessionToken, _ := uuid.NewV4()

	ss.sessions[sessionToken.String()] = s.Payload

	s.Token = sessionToken.String()
	return nil
}

// Delete удаляет сессию с токен s.Token из хранилища
func (ss *SessionsTest) Delete(s *models.Session) error {
	if err := checkFailureSession(); err != nil {
		return err
	}
	delete(ss.sessions, s.Token)

	return nil
}

// GetSession получает сессию из хранилища по токену
func (ss *SessionsTest) GetSession(token string) (*models.Session, error) {
	if err := checkFailureSession(); err != nil {
		return nil, err
	}
	data, ok := ss.sessions[token]
	if !ok {
		return nil, errors.Wrap(errors.New("not found"), "redis get error")
	}

	return &models.Session{
		Token:   token,
		Payload: data,
	}, nil
}

type Case struct {
	Payload        []byte
	ExpectedCode   int
	ExpectedBody   string
	Method         string
	Endpoint       string
	Pattern        string
	Vars           map[string]string
	Function       http.HandlerFunc
	Context        context.Context
	FailureUser    error
	FailureSession error
}

func initTests() {
	curID = 0
	models.Users = &UsersTest{
		users:    make(map[int64]models.User),
		nextFail: nil,
	}

	models.Sessions = &SessionsTest{
		sessions: make(map[string][]byte),
		nextFail: nil,
	}
}

func makeRequest(ctx context.Context, handler http.Handler, method, endpoint string, cookies []*http.Cookie,
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

func runAPITest(t *testing.T, i int, c *Case) {
	setFailureUser(c.FailureUser)
	setFailureSession(c.FailureSession)

	r := mux.NewRouter()
	r.HandleFunc(c.Pattern, c.Function).Methods(c.Method)
	if c.Endpoint == "" {
		c.Endpoint = c.Pattern
	}
	resp := makeRequest(c.Context, r, c.Method, c.Endpoint,
		nil, bytes.NewBuffer(c.Payload))
	if resp.Code != c.ExpectedCode {
		t.Fatalf("\n[%d] Expected response code %d Got %d\n\n[%d] Expected response:\n %s\n Got:\n %s\n", i, c.ExpectedCode, resp.Code, i, c.ExpectedBody, resp.Body.String())
	}

	if resp.Body.String() != c.ExpectedBody {
		t.Fatalf("\n[%d] Expected response:\n %s\n Got:\n %s\n", i, c.ExpectedBody, resp.Body.String())
	}
}

func runTableAPITests(t *testing.T, cases []*Case) {
	for i, c := range cases {
		runAPITest(t, i, c)
	}
}

func TestPack(t *testing.T) {

	initTests()
	var expErr = errors.New("test failure")
	setFailureUser(expErr)
	err := models.Users.Create(&models.User{})
	if err != expErr {
		t.Fatalf("test failure failed %s)))))))))", err.Error())
	}
}

func TestCreateUser(t *testing.T) {
	initTests()

	cases := []*Case{
		{ //Всё ок
			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
			ExpectedCode: 200,
			ExpectedBody: ``,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
		},
		{ // На используемый username
			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
			ExpectedCode: 400,
			ExpectedBody: `{"username":"taken"}`,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
			FailureUser:  models.ErrUsernameTaken,
		},
		{ // Пустой юзернейм
			Payload:      []byte(`{"username":"","password":"dsadasd"}`),
			ExpectedCode: 400,
			ExpectedBody: `{"username":"required"}`,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
		},
		{ // Пустой пароль нас не смущает  TODO: сделать пустой пароль более развратным
			Payload:      []byte(`{"username":"kek","password":""}`),
			ExpectedCode: 200,
			ExpectedBody: ``,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
		},
		{ // Неправильный формат JSON
			Payload:      []byte(`{"username":"kek""}`),
			ExpectedCode: 400,
			ExpectedBody: `{"message":"decode body error: invalid character '\"' after object key:value pair"}`,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
		},
	}

	runTableAPITests(t, cases)
}

func TestUpdateUser(t *testing.T) {
	initTests()

	cases := []*Case{
		{ //Такого юзера пока нет
			Payload:      []byte(`{"username":"kek","password":"lol"}`),
			ExpectedCode: 401,
			ExpectedBody: `{"message":"user not exists: get user error: not_exists"}`,
			Method:       "PUT",
			Pattern:      "/users",
			Function:     UpdateUser,
			Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{}),
		},
		{ // Создадим юзера
			Payload:      []byte(`{"username":"kek","password":"lol"}`),
			ExpectedCode: 200,
			ExpectedBody: ``,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
			Context:      context.Background(),
		},
		{ // Пустой никнейм нельзя
			Payload:      []byte(`{"username":"", "password":"lol"}`),
			ExpectedCode: 400,
			ExpectedBody: `{"username":"invalid"}`,
			Method:       "PUT",
			Pattern:      "/users",
			Function:     UpdateUser,
			Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1}),
		},
		{ // Неправильный формат JSON
			Payload:      []byte(`{"username":"kek""}`),
			ExpectedCode: 400,
			ExpectedBody: `{"message":"decode body error: invalid character '\"' after object key:value pair"}`,
			Method:       "PUT",
			Pattern:      "/users",
			Function:     UpdateUser,
			Context:      context.Background(),
		},
		{
			Payload:      []byte(`{"username":"kek", "newPassword":"lol"}`),
			ExpectedCode: 400,
			ExpectedBody: `{"oldPassword":"required"}`,
			Method:       "PUT",
			Pattern:      "/users",
			Function:     UpdateUser,
			Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1}),
		},
		{
			Payload:      []byte(`{"username":"kek", "oldPassword":"hh", "newPassword":"lol"}`),
			ExpectedCode: 400,
			ExpectedBody: `{"oldPassword":"invalid"}`,
			Method:       "PUT",
			Pattern:      "/users",
			Function:     UpdateUser,
			Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1}),
		},
		{
			Payload:      []byte(`{"username":"kek", "photo_uuid":"ne photoUUID"}`),
			ExpectedCode: 400,
			ExpectedBody: `{"photo_uuid":"invalid"}`,
			Method:       "PUT",
			Pattern:      "/users",
			Function:     UpdateUser,
			Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1}),
		},
		{
			Payload:      []byte(`{"username":"kek", "oldPassword":"lol", "newPassword":"lol1", "photo_uuid":"2eb4a823-3a6d-4cba-8767-4d4946890f4f"}`),
			ExpectedCode: 200,
			ExpectedBody: ``,
			Method:       "PUT",
			Pattern:      "/users",
			Function:     UpdateUser,
			Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1}),
		},
	}

	runTableAPITests(t, cases)
}

func TestCheckUsername(t *testing.T) {
	initTests()

	cases := []*Case{
		{ //Всё ок
			Payload:      []byte(`{"username":"sdas"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"used":false}`,
			Method:       "POST",
			Pattern:      "/users/username_check",
			Function:     CheckUsername,
		},
		{ // Создадим юзера
			Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
			ExpectedCode: 200,
			ExpectedBody: ``,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
		},
		{ // Теперь уже имя занято
			Payload:      []byte(`{"username":"sdas"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"used":true}`,
			Method:       "POST",
			Pattern:      "/users/used",
			Function:     CheckUsername,
		},
		{ // Пустой никнейм, очевидно, свободен, но зарегать его всё равно нельзя
			Payload:      []byte(`{"username":""}`),
			ExpectedCode: 200,
			ExpectedBody: `{"used":false}`,
			Method:       "POST",
			Pattern:      "/users/used",
			Function:     CheckUsername,
		},
		{ // Неправильный формат JSON
			Payload:      []byte(`{"username":"kek""}`),
			ExpectedCode: 400,
			ExpectedBody: `{"message":"decode body error: invalid character '\"' after object key:value pair"}`,
			Method:       "POST",
			Pattern:      "/users/used",
			Function:     CheckUsername,
		},
	}

	runTableAPITests(t, cases)
}

func TestGetUser(t *testing.T) {
	initTests()

	cases := []*Case{
		{ // Такого юзера пока нет
			ExpectedCode: 404,
			ExpectedBody: `{"message":"user not exists: not_exists"}`,
			Method:       "GET",
			Pattern:      "/users/{user_id:[0-9]+}",
			Endpoint:     "/users/1",
			Vars:         map[string]string{"user_id": "1"},
			Function:     GetUser,
		},
		{ // Создадим юзера
			Payload:      []byte(`{"username":"golang","password":"4ever"}`),
			ExpectedCode: 200,
			ExpectedBody: ``,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
		},
		{ //Всё ок
			ExpectedCode: 200,
			ExpectedBody: `{"username":"golang","photo_uuid":"","id":1,"active":true}`,
			Method:       "GET",
			Pattern:      "/users/{user_id:[0-9]+}",
			Endpoint:     "/users/1",
			Function:     GetUser,
		},
	}

	runTableAPITests(t, cases)
}

func TestSession(t *testing.T) {
	initTests()

	cases := []*Case{
		{
			Payload:      []byte(`{"username":"golang","password":"4ever"}`),
			ExpectedCode: 200,
			ExpectedBody: ``,
			Method:       "POST",
			Pattern:      "/users",
			Function:     CreateUser,
		},
		{
			Payload:      []byte(`{"username":"golang","password":"4ever"}`),
			ExpectedCode: 200,
			ExpectedBody: ``,
			Method:       "POST",
			Pattern:      "/sessions",
			Function:     CreateSession,
		},
		{
			Payload:      []byte(`{"username":"golang","password":"4ever"}`),
			ExpectedCode: 200,
			ExpectedBody: `{"username":"golang","photo_uuid":"","id":1,"active":true}`,
			Method:       "POST",
			Pattern:      "/sessions",
			Function:     GetSession,
			Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1}),
		},
		{
			Payload:      []byte(``),
			ExpectedCode: 401,
			ExpectedBody: `{"message":"get cookie error: http: named cookie not present"}`,
			Method:       "DELETE",
			Pattern:      "/sessions",
			Function:     DeleteSession,
		},
	}

	runTableAPITests(t, cases)
}
