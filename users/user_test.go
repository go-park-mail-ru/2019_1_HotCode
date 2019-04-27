package users

import (
	"context"
	"net/http"
	"testing"

	"github.com/HotCodeGroup/warscript-utils/utils"

	"github.com/go-park-mail-ru/2019_1_HotCode/testutils"

	"github.com/jackc/pgx/pgtype"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

func init() {
	// чтобы не заваливать всё логами
	log.SetLevel(log.PanicLevel)
}

type UsersTest struct {
	ids      int64
	users    map[int64]UserModel
	nextFail error
}

func (ut *UsersTest) newID() int64 {
	ut.ids++
	return ut.ids - 1
}

// setFailureUser fails next request
func setFailureUser(err error) {
	us := Users.(*UsersTest)
	us.nextFail = err
}

func checkFailureUser() error {
	us := Users.(*UsersTest)
	if us.nextFail != nil {
		defer func() {
			us.nextFail = nil
		}()
		return us.nextFail
	}
	return nil
}

// Create создаёт запись в базе с новыми полями
func (ut *UsersTest) Create(u *UserModel) error {
	if err := checkFailureUser(); err != nil {
		return err
	}
	u.Active = pgtype.Bool{Bool: true, Status: pgtype.Present}
	u.ID = pgtype.Int8{Int: ut.newID(), Status: pgtype.Present}
	ut.users[u.ID.Int] = *u
	return nil
}

// Save сохраняет юзера в базу
func (ut *UsersTest) Save(u *UserModel) error {
	if err := checkFailureUser(); err != nil {
		return err
	}

	ut.users[u.ID.Int] = *u
	return nil
}

// CheckPassword проверяет пароль у юзера и сохранённый в модели
func (ut *UsersTest) CheckPassword(u *UserModel, password string) bool {
	return *u.Password == password
}

// GetUserByID получает юзера по id
func (ut *UsersTest) GetUserByID(id int64) (*UserModel, error) {
	if err := checkFailureUser(); err != nil {
		return nil, err
	}
	var u UserModel
	var ok bool
	if u, ok = ut.users[id]; !ok {
		return nil, utils.ErrNotExists
	}

	return &u, nil
}

// GetUserByUsername получает юзера по имени
func (ut *UsersTest) GetUserByUsername(username string) (*UserModel, error) {
	if err := checkFailureUser(); err != nil {
		return nil, err
	}
	var u UserModel
	var ok bool

	for _, user := range ut.users {
		if user.Username.String == username {
			ok = true
			u = user
		}
	}
	if !ok {
		return nil, utils.ErrNotExists
	}

	return &u, nil
}

type SessionsTest struct {
	sessions map[string][]byte
	nextFail error
}

func setFailureSession(err error) {
	ss := Sessions.(*SessionsTest)
	ss.nextFail = err
}

func checkFailureSession() error {
	ss := Sessions.(*SessionsTest)
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
func (ss *SessionsTest) Set(s *Session) error {
	if err := checkFailureSession(); err != nil {
		return err
	}

	ss.sessions[s.Token] = s.Payload
	return nil
}

// Delete удаляет сессию с токен s.Token из хранилища
func (ss *SessionsTest) Delete(s *Session) error {
	if err := checkFailureSession(); err != nil {
		return err
	}
	delete(ss.sessions, s.Token)

	return nil
}

// GetSession получает сессию из хранилища по токену
func (ss *SessionsTest) GetSession(token string) (*Session, error) {
	if err := checkFailureSession(); err != nil {
		return nil, err
	}
	data, ok := ss.sessions[token]
	if !ok {
		return nil, errors.Wrap(errors.New("not found"), "redis get error")
	}

	return &Session{
		Token:   token,
		Payload: data,
	}, nil
}

func initTests() {
	Users = &UsersTest{
		ids:      1,
		users:    make(map[int64]UserModel),
		nextFail: nil,
	}

	Sessions = &SessionsTest{
		sessions: make(map[string][]byte),
		nextFail: nil,
	}
}

type UserTestCase struct {
	testutils.Case
	FailureUser    error
	FailureSession error
}

func runTableAPITests(t *testing.T, cases []*UserTestCase) {
	for i, c := range cases {
		runAPITest(t, i, c)
	}
}

func runAPITest(t *testing.T, i int, c *UserTestCase) {
	if c.FailureUser != nil {
		setFailureUser(c.FailureUser)
	}

	if c.FailureSession != nil {
		setFailureSession(c.FailureSession)
	}

	testutils.RunAPITest(t, i, &c.Case)
}

func TestCreateUser(t *testing.T) {
	initTests()

	cases := []*UserTestCase{
		{ // Всё ок
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
		},
		{ // На используемый username
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"username":"taken"}`,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
			FailureUser: utils.ErrTaken,
		},
		{ // Пустой юзернейм
			Case: testutils.Case{
				Payload:      []byte(`{"username":"","password":"dsadasd"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"username":"required"}`,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
		},
		{ // Пустой пароль теперь нас очень смущает
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek","password":""}`),
				ExpectedCode: 400,
				ExpectedBody: `{"password":"required"}`,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
		},
		{ // Неправильный формат JSON
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek""}`),
				ExpectedCode: 400,
				ExpectedBody: `{"message":"decode body error: invalid character '\"' after object key:value pair"}`,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
		},
		{ // Упала база
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
				ExpectedCode: 500,
				ExpectedBody: `{"message":"user create error: internal server error"}`,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
			FailureUser: utils.ErrInternal,
		},
		{ // По какой-то невообразимой причине только что созданный юзер не существует в базе
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
				ExpectedCode: 500,
				ExpectedBody: `{"message":"set session error: map[lol:kek]"}`,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
			FailureSession: &utils.ValidationError{"lol": "kek"},
		},
		{ // Редис упал
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
				ExpectedCode: 500,
				ExpectedBody: `{"message":"set session error: internal server error"}`,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
			FailureSession: utils.ErrInternal,
		},
	}

	runTableAPITests(t, cases)
}

func TestUpdateUser(t *testing.T) {
	initTests()

	cases := []*UserTestCase{
		{ // Такого юзера пока нет
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek","password":"lol"}`),
				ExpectedCode: 401,
				ExpectedBody: `{"message":"user not exists: get user error: not_exists"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{}),
			},
		},
		{ // Создадим юзера
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek","password":"lol"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
				Context:      context.Background(),
			},
		},
		{ // Пустой никнейм нельзя
			Case: testutils.Case{
				Payload:      []byte(`{"username":"", "password":"lol"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"username":"invalid"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{ // Неправильный формат JSON
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek""}`),
				ExpectedCode: 400,
				ExpectedBody: `{"message":"decode body error: invalid character '\"' after object key:value pair"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{ // Нет контекста
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek""}`),
				ExpectedCode: 401,
				ExpectedBody: `{"message":"session info is not presented"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.Background(),
			},
		},
		{ // нельзя поставить путой пароль
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek", "oldPassword":"hh", "newPassword":""}`),
				ExpectedCode: 400,
				ExpectedBody: `{"newPassword":"invalid"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek", "newPassword":"lol"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"oldPassword":"required"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek", "oldPassword":"hh", "newPassword":"lol"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"oldPassword":"invalid"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek", "photo_uuid":"ne photoUUID"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"photo_uuid":"invalid"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{
			Case: testutils.Case{
				Payload: []byte(`{"username":"kek", "oldPassword":"lol", "newPassword":"lol1",
			 					"photo_uuid":"2eb4a823-3a6d-4cba-8767-4d4946890f4f"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{ // отвалилась база
			Case: testutils.Case{
				Payload: []byte(`{"username":"kek", "oldPassword":"lol", "newPassword":"lol1",
			 					"photo_uuid":"2eb4a823-3a6d-4cba-8767-4d4946890f4f"}`),
				ExpectedCode: 500,
				ExpectedBody: `{"message":"get user error: upala basa"}`,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
			FailureUser: errors.New("upala basa"),
		},
		{ // нечего обновлять
			Case: testutils.Case{
				Payload:      []byte(`{}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
	}

	runTableAPITests(t, cases)
}

func TestCheckUsername(t *testing.T) {
	initTests()

	cases := []*UserTestCase{
		{ // Всё ок
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas"}`),
				ExpectedCode: 200,
				ExpectedBody: `{"used":false}`,
				Method:       "POST",
				Pattern:      "/users/username_check",
				Function:     CheckUsername,
			},
		},
		{ // Создадим юзера
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas","password":"dsadasd"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
		},
		{ // Теперь уже имя занято
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas"}`),
				ExpectedCode: 200,
				ExpectedBody: `{"used":true}`,
				Method:       "POST",
				Pattern:      "/users/used",
				Function:     CheckUsername,
			},
		},
		{ // Пустой никнейм, очевидно, свободен, но зарегать его всё равно нельзя
			Case: testutils.Case{
				Payload:      []byte(`{"username":""}`),
				ExpectedCode: 200,
				ExpectedBody: `{"used":false}`,
				Method:       "POST",
				Pattern:      "/users/used",
				Function:     CheckUsername,
			},
		},
		{ // Неправильный формат JSON
			Case: testutils.Case{
				Payload:      []byte(`{"username":"kek""}`),
				ExpectedCode: 400,
				ExpectedBody: `{"message":"decode body error: invalid character '\"' after object key:value pair"}`,
				Method:       "POST",
				Pattern:      "/users/used",
				Function:     CheckUsername,
			},
		},
		{ // Отвалилась база
			Case: testutils.Case{
				Payload:      []byte(`{"username":"sdas"}`),
				ExpectedCode: 500,
				ExpectedBody: `{"message":"get user method error: upala basa"}`,
				Method:       "POST",
				Pattern:      "/users/used",
				Function:     CheckUsername,
			},
			FailureUser: errors.New("upala basa"),
		},
	}

	runTableAPITests(t, cases)
}

func TestGetUser(t *testing.T) {
	initTests()

	cases := []*UserTestCase{
		{ // Такого юзера пока нет
			Case: testutils.Case{
				ExpectedCode: 404,
				ExpectedBody: `{"message":"user not exists: not_exists"}`,
				Method:       "GET",
				Pattern:      "/users/{user_id:[0-9]+}",
				Endpoint:     "/users/1",
				Function:     GetUser,
			},
		},
		{ // user_id в неверном формате(выключаем встроенную фильтрацию от gorilla)
			Case: testutils.Case{
				ExpectedCode: 404,
				ExpectedBody: `{"message":"wrong format user_id: strconv.ParseInt: parsing \"keks\": invalid syntax"}`,
				Method:       "GET",
				Pattern:      "/users/{user_id}",
				Endpoint:     "/users/keks",
				Function:     GetUser,
			},
		},
		{ // Создадим юзера
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang","password":"4ever"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
		},
		{ // добавили авку
			Case: testutils.Case{
				Payload:      []byte(`{"photo_uuid":"2eb4a823-3a6d-4cba-8767-4d4946890f4f"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{ // Всё ок
			Case: testutils.Case{
				ExpectedCode: 200,
				ExpectedBody: `{"username":"golang","photo_uuid":"2eb4a823-3a6d-4cba-8767-4d4946890f4f","id":1,"active":true}`,
				Method:       "GET",
				Pattern:      "/users/{user_id:[0-9]+}",
				Endpoint:     "/users/1",
				Function:     GetUser,
			},
		},
		{ // Упала база
			Case: testutils.Case{
				ExpectedCode: 500,
				ExpectedBody: `{"message":"get user method error: upala basa"}`,
				Method:       "GET",
				Pattern:      "/users/{user_id:[0-9]+}",
				Endpoint:     "/users/1",
				Function:     GetUser,
			},
			FailureUser: errors.New("upala basa"),
		},
	}

	runTableAPITests(t, cases)
}

func TestCreateSession(t *testing.T) {
	initTests()

	cases := []*UserTestCase{
		{ // кривой JSON(без запятой)
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang" "password":"4ever"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"message":"decode body error: invalid character '\"' after object key:value pair"}`,
				Method:       "POST",
				Pattern:      "/sessions",
				Function:     CreateSession,
			},
		},
		{ // без пароля
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"password":"required"}`,
				Method:       "POST",
				Pattern:      "/sessions",
				Function:     CreateSession,
			},
		},
		{ // незареганный юзер
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang", "password":"4ever"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"username":"not_exists"}`,
				Method:       "POST",
				Pattern:      "/sessions",
				Function:     CreateSession,
			},
		},
		// зарегали юзера
		{
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang","password":"4ever"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
		},
		{ // неправильный пароль
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang", "password":"4ever2312123"}`),
				ExpectedCode: 400,
				ExpectedBody: `{"password":"invalid"}`,
				Method:       "POST",
				Pattern:      "/sessions",
				Function:     CreateSession,
			},
		},
		{ // Отломалось хранилище сессий
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang","password":"4ever"}`),
				ExpectedCode: 500,
				ExpectedBody: `{"message":"set session error: vse slomalos"}`,
				Method:       "POST",
				Pattern:      "/sessions",
				Function:     CreateSession,
			},
			FailureSession: errors.New("vse slomalos"),
		},
		{ // Всё ок
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang","password":"4ever"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "POST",
				Pattern:      "/sessions",
				Function:     CreateSession,
			},
		},
	}

	runTableAPITests(t, cases)
}

func TestDeleteSession(t *testing.T) {
	initTests()

	cases := []*UserTestCase{
		{ // без куки совсем
			Case: testutils.Case{
				ExpectedCode: 401,
				ExpectedBody: `{"message":"get cookie error: http: named cookie not present"}`,
				Method:       "DELETE",
				Pattern:      "/sessions",
				Function:     DeleteSession,
			},
		},
		{ // Отвалился storage
			Case: testutils.Case{
				ExpectedCode: 500,
				ExpectedBody: `{"message":"session delete error: storage upal"}`,
				Method:       "POST",
				Pattern:      "/sessions",
				Cookies: []*http.Cookie{
					{
						Name:  "JSESSIONID",
						Value: "12345",
					},
				},
				Function: DeleteSession,
			},
			FailureSession: errors.New("storage upal"),
		},
		{ // без пароля
			Case: testutils.Case{
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "POST",
				Pattern:      "/sessions",
				Cookies: []*http.Cookie{
					{
						Name:  "JSESSIONID",
						Value: "12345",
					},
				},
				Function: DeleteSession,
			},
		},
	}

	runTableAPITests(t, cases)
}

func TestGetSession(t *testing.T) {
	initTests()

	cases := []*UserTestCase{
		{ // без куки совсем
			Case: testutils.Case{
				ExpectedCode: 401,
				ExpectedBody: `{"message":"session info is not presented"}`,
				Method:       "DELETE",
				Pattern:      "/sessions",
				Function:     GetSession,
			},
		},
		{ // несуществующий юзер
			Case: testutils.Case{
				ExpectedCode: 401,
				ExpectedBody: `{"message":"user not exists: not_exists"}`,
				Method:       "DELETE",
				Pattern:      "/sessions",
				Function:     GetSession,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{ // упала база
			Case: testutils.Case{
				ExpectedCode: 500,
				ExpectedBody: `{"message":"basa upala"}`,
				Method:       "DELETE",
				Pattern:      "/sessions",
				Function:     GetSession,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
			FailureUser: errors.New("basa upala"),
		},
		{ // зарегали юзера
			Case: testutils.Case{
				Payload:      []byte(`{"username":"golang","password":"4ever"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "POST",
				Pattern:      "/users",
				Function:     CreateUser,
			},
		},
		{ // добавили авку
			Case: testutils.Case{
				Payload:      []byte(`{"photo_uuid":"2eb4a823-3a6d-4cba-8767-4d4946890f4f"}`),
				ExpectedCode: 200,
				ExpectedBody: ``,
				Method:       "PUT",
				Pattern:      "/users",
				Function:     UpdateUser,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
		{ // теперь всё ок
			Case: testutils.Case{
				ExpectedCode: 200,
				ExpectedBody: `{"username":"golang","photo_uuid":"2eb4a823-3a6d-4cba-8767-4d4946890f4f","id":1,"active":true}`,
				Method:       "DELETE",
				Pattern:      "/sessions",
				Function:     GetSession,
				Context:      context.WithValue(context.Background(), SessionInfoKey, &SessionPayload{1, 1}),
			},
		},
	}

	runTableAPITests(t, cases)
}

func TestMiddleware(t *testing.T) {
	initTests()

	testFunction := func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("kek"))
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}

	cases := []*UserTestCase{
		{ // без куки совсем
			Case: testutils.Case{
				ExpectedCode: 401,
				ExpectedBody: `{"message":"can not load cookie: http: named cookie not present"}`,
				Method:       "GET",
				Pattern:      "/test",
				Function:     WithAuthentication(testFunction),
			},
		},
		{ // неверный куки
			Case: testutils.Case{
				ExpectedCode: 401,
				ExpectedBody: `{"message":"get session error: redis get error: not found"}`,
				Method:       "GET",
				Pattern:      "/test",
				Cookies: []*http.Cookie{
					{
						Name:  "JSESSIONID",
						Value: "12345",
					},
				},
				Function: WithAuthentication(testFunction),
			},
		},
	}

	runTableAPITests(t, cases)

	err := Sessions.Set(&Session{
		Token:   "kek",
		Payload: []byte("kek"),
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	err = Sessions.Set(&Session{
		Token:   "kek1",
		Payload: []byte(`{"kek": "kek"}`),
	})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	cases1 := []*UserTestCase{
		{ // в редисе кривой JSON
			Case: testutils.Case{
				ExpectedCode: 500,
				ExpectedBody: `{"message":"session payload unmarshal error: invalid character 'k' looking for beginning of value"}`,
				Method:       "GET",
				Pattern:      "/test",
				Cookies: []*http.Cookie{
					{
						Name:  "JSESSIONID",
						Value: "kek",
					},
				},
				Function: WithAuthentication(testFunction),
			},
		},
		{ // Всё ок
			Case: testutils.Case{
				ExpectedCode: 200,
				ExpectedBody: `kek`,
				Method:       "GET",
				Pattern:      "/test",
				Cookies: []*http.Cookie{
					{
						Name:  "JSESSIONID",
						Value: "kek1",
					},
				},
				Function: WithAuthentication(testFunction),
			},
		},
	}

	runTableAPITests(t, cases1)
}
