package dblib

import (
<<<<<<< HEAD
	"2019_1_HotCode/apptypes"
=======
	errCodes "Techno/2019_1_HotCode/errors"
	"errors"
>>>>>>> f7d6d7f1aca1f8608e5f353094286626ac23f23f
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
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

func initDB() {
	ConnectDB("warscript_test_user", "qwerty",
		"localhost", "warscript_test_db")
	GetDB().Exec(reloadTableSQL)
	GetStorage().Do("FLUSHDB")
}

func FormErrorsToString(fe *apptypes.Errors) string {
	if fe == nil {
		return "	nil\n"
	}
	var sb strings.Builder
	sb.WriteString("	Errors:\n")
	for name, err := range fe.Fields {
		sb.WriteString(fmt.Sprintf(`		"%s":{
		Code: %d
		Descr: %s
		Msg: %s
	}
`, name, err.Code, err.Description, err.Message))
	}
	sb.WriteString("	Other:\n")
	sb.WriteString(fmt.Sprintf(`
		Code: %d
		Descr: %s
		Msg: %s
`, fe.Other.Code, fe.Other.Description, fe.Other.Message))

	return sb.String()
}

//Create Save Validate User Case
type CSVUserCase struct {
	u    *User
	resp *apptypes.Errors
}

func TestUserCreate(t *testing.T) {
	user := &User{
		Username: "another_user",
		Password: []byte("secure_password"),
	}
	cases := []*CSVUserCase{
		//1
		&CSVUserCase{
			u: &User{
				Username: "test_user",
				Password: []byte("secure_password"),
			},
			resp: nil,
		},

		//2
		&CSVUserCase{
			u: &User{
				Username: "test_user",
				Password: []byte("secure_password"),
			},
			resp: &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"username": &apptypes.Error{
						Code:        apptypes.AlreadyUsed,
						Description: `pq: duplicate key value violates unique constraint "uniq_username"`,
					},
				},
			},
		},

		//3
		&CSVUserCase{
			u:    user,
			resp: nil,
		},

		//4
		&CSVUserCase{
			u: user,
			resp: &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"id": &apptypes.Error{
						Code:        apptypes.CantCreate,
						Description: `User cant be formed, maybe try Update?`,
					},
				},
			},
		},
	}
	initDB()
	for i, c := range cases {
		if resp := c.u.Create(); !reflect.DeepEqual(resp, c.resp) {
			t.Errorf("\nUser Create test %d failed: \nEXPECTED: \n%sGOT: \n%s", i+1, FormErrorsToString(c.resp), FormErrorsToString(resp))
		}
	}
}

func TestUserSave(t *testing.T) {
	initDB()
	user1 := &User{
		Username: "user1",
		Password: []byte("secure_password"),
	}
	user1.Create()
	user2 := &User{
		Username: "user2",
		Password: []byte("secure_password"),
	}
	user2.Create()
	user1.Password = []byte("very_secure_password")
	user2.Username = "user1"
	cases := []*CSVUserCase{
		//1
		&CSVUserCase{
			u:    user1,
			resp: nil,
		},

		//2
		&CSVUserCase{
			u: user2,
			resp: &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"username": &apptypes.Error{
						Code:        apptypes.AlreadyUsed,
						Description: `pq: duplicate key value violates unique constraint "uniq_username"`,
					},
				},
			},
		},

		//3
		&CSVUserCase{
			u: &User{
				Username: "test_user",
				Password: []byte("secure_password"),
			},
			resp: &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"id": &apptypes.Error{
						Code:        apptypes.CantSave,
						Description: `User doesn't exist, maybe try Create?`,
					},
				},
			},
		},
	}
	initDB()
	for i, c := range cases {
		if resp := c.u.Save(); !reflect.DeepEqual(resp, c.resp) {
			t.Errorf("\nUser Save test %d failed: \nEXPECTED: \n%sGOT: \n%s", i+1, FormErrorsToString(c.resp), FormErrorsToString(resp))
		}
	}
}

func TestUserValidate(t *testing.T) {
	initDB()
	cases := []*CSVUserCase{
		//1
		&CSVUserCase{
			u: &User{
				Username: "has",
				Password: []byte("has"),
			},
			resp: nil,
		},

		//2
		&CSVUserCase{
			u: &User{
				Username: "",
				Password: []byte("has"),
			},
			resp: &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"username": &apptypes.Error{
						Code:        apptypes.FailedToValidate,
						Description: "Username is empty",
					},
				},
			},
		},

		//3
		&CSVUserCase{
			u: &User{
				Username: "has",
				Password: []byte(""),
			},
			resp: &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"password": &apptypes.Error{
						Code:        apptypes.FailedToValidate,
						Description: `Password is empty`,
					},
				},
			},
		},

		//4
		&CSVUserCase{
			u: &User{
				Username: "",
				Password: []byte(""),
			},
			resp: &apptypes.Errors{
				Fields: map[string]*apptypes.Error{
					"password": &apptypes.Error{
						Code:        apptypes.FailedToValidate,
						Description: `Password is empty`,
					},
					"username": &apptypes.Error{
						Code:        apptypes.FailedToValidate,
						Description: "Username is empty",
					},
				},
			},
		},
	}
	initDB()
	for i, c := range cases {
		if resp := c.u.Validate(); !reflect.DeepEqual(resp, c.resp) {
			t.Errorf("\nUser Validate test %d failed: \nEXPECTED: \n%sGOT: \n%s", i+1, FormErrorsToString(c.resp), FormErrorsToString(resp))
		}
	}
}

type GetUserCase struct {
	params map[string]interface{}
	u      *User
	resp   *apptypes.Errors
}

func TestUserGet(t *testing.T) {
	initDB()
	user1 := &User{
		Username: "user1",
		Password: []byte("secure_password"),
	}
	user1.Create()
	user2 := &User{
		Username: "user2",
		Password: []byte("secure_password"),
	}
	user2.Create()
	cases := []*GetUserCase{
		//1
		&GetUserCase{
			params: map[string]interface{}{
				"username": "not_exist",
			},
			u: nil,
			resp: &apptypes.Errors{
				Other: &apptypes.Error{
					Code:        apptypes.RowNotFound,
					Description: "Can't find user with given parameters",
				},
			},
		},

		//2
		&GetUserCase{
			params: map[string]interface{}{
				"username": "user1",
			},
			u:    user1,
			resp: nil,
		},

		//3
		&GetUserCase{
			params: map[string]interface{}{
				"id": "2",
			},
			u:    user2,
			resp: nil,
		},

		//4
		&GetUserCase{
			params: map[string]interface{}{
				"speed": "500",
			},
			u: nil,
			resp: &apptypes.Errors{
				Other: &apptypes.Error{
					Code:        apptypes.RowNotFound,
					Description: "Can't find user with given parameters",
				},
			},
		},
	}
	for i, c := range cases {
		if u, resp := GetUser(c.params); !reflect.DeepEqual(resp, c.resp) || !reflect.DeepEqual(u, c.u) {
			t.Errorf("\nUser GetUser test %d failed: \nEXPECTED: \n%s\n%vGOT: \n%s\n%v", i+1, FormErrorsToString(c.resp), c.u, FormErrorsToString(resp), u)
		}
	}
}

//Save Get Del Storage Case
type SGDStorageCase struct {
	k        string
	v        []byte
	t        time.Duration
	method   interface{}
	response SGDStorageCaseResp
}

type SGDStorageCaseResp struct {
	v   []byte
	err error
}

func errEq(e1 error, e2 error) bool {
	return (e1 == nil && e2 == nil) ||
		(e1 != nil && e2 != nil && e1.Error() == e2.Error())
}

func TestStorageSGD(t *testing.T) {
	cases := []SGDStorageCase{
		//1
		SGDStorageCase{
			k:      "key",
			v:      []byte("value"),
			t:      1000 * time.Second,
			method: "SET",
			response: SGDStorageCaseResp{
				err: nil,
			},
		},

		//2
		SGDStorageCase{
			k:      "key",
			method: "GET",
			response: SGDStorageCaseResp{
				err: nil,
				v:   []byte("value"),
			},
		},

		//3
		SGDStorageCase{
			k:      "key",
			v:      []byte("another value"),
			t:      1000 * time.Second,
			method: "SET",
			response: SGDStorageCaseResp{
				err: nil,
			},
		},

		//4
		SGDStorageCase{
			k:      "key",
			method: "GET",
			response: SGDStorageCaseResp{
				err: nil,
				v:   []byte("another value"),
			},
		},

		//5
		SGDStorageCase{
			k:      "key",
			method: "DEL",
			response: SGDStorageCaseResp{
				err: nil,
			},
		},

		//6
		SGDStorageCase{
			k:      "key",
			method: "GET",
			response: SGDStorageCaseResp{
				err: errors.New("failed to get value: redigo: nil returned"),
				v:   nil,
			},
		},

		//7
		SGDStorageCase{
			k:      "another_key",
			method: "DEL",
			response: SGDStorageCaseResp{
				err: nil,
			},
		},
	}

	for i, c := range cases {
		switch c.method {
		//StorageSet
		case "SET":
			if err := StorageSet(c.k, c.v, c.t); err != c.response.err {
				t.Errorf("\nSGD test %d failed:\nEXPECTED: %v\nGOT: %v", i+1, c.response.err, err)
			}
		//StorageGet
		case "GET":
			if v, err := StorageGet(c.k); !errEq(err, c.response.err) || !reflect.DeepEqual(v, c.response.v) {
				t.Errorf("\nSGD test %d failed:\nEXPECTED: (%s,%v)\nGOT: (%s,%v)", i+1, c.response.v, c.response.err, v, err)
			}
		//StorageDel
		case "DEL":
			if err := StorageDel(c.k); err != c.response.err {
				t.Errorf("\nSGD test %d failed:\nEXPECTED: %v\nGOT: %v", i+1, c.response.err, err)
			}
		}
	}
}
