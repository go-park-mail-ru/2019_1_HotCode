package controllers

import (
	"testing"

	"github.com/go-park-mail-ru/2019_1_HotCode/models"

	"github.com/pkg/errors"
)

type UsersTest struct {
	users    map[int64]models.User
	nextFail error
}

// setFailure fails next request
func setFailure(err error) {
	us := models.Users.(*UsersTest)
	us.nextFail = err
}

func checkFailure() error {
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
	if err := checkFailure(); err != nil {
		return err
	}

	u.ID = newID()
	us.users[u.ID] = *u
	return nil
}

// Save сохраняет юзера в базу
func (us *UsersTest) Save(u *models.User) error {
	if err := checkFailure(); err != nil {
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
	if err := checkFailure(); err != nil {
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
	if err := checkFailure(); err != nil {
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

func initTests() {
	models.Users = &UsersTest{
		users:    make(map[int64]models.User),
		nextFail: nil,
	}
}

func TestPack(t *testing.T) {
	initTests()
	var expErr = errors.New("test failure")
	setFailure(expErr)
	err := models.Users.Create(&models.User{})
	if err != expErr {
		t.Fatalf("test failure failed %s)))))))))", err.Error())
	}
}
