package dblib

import (
	errCodes "Techno/2019_1_HotCode/errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
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
	GetDB().Exec(reloadTableSQL)
}

func FormErrorsToString(fe *FormErrors) string {
	if fe == nil {
		return "	nil\n"
	}
	var sb strings.Builder
	sb.WriteString("	Errors:\n")
	for name, err := range fe.Errors {
		sb.WriteString(fmt.Sprintf(`		"%s":{
		Code: %d
		Descr: %s
		Msg: %s
	}
`, name, err.Code, err.Description, err.Message))
	}
	sb.WriteString("	Other:\n")
	for _, err := range fe.Other {
		sb.WriteString(fmt.Sprintf(`
		Code: %d
		Descr: %s
		Msg: %s
`, err.Code, err.Description, err.Message))
	}
	return sb.String()
}

//Create Save Validate User Case
type CSVUserCase struct {
	u    *User
	resp *FormErrors
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"username": &Error{
						Code:        errCodes.AlreadyUsed,
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"id": &Error{
						Code:        errCodes.CantCreate,
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"username": &Error{
						Code:        errCodes.AlreadyUsed,
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"id": &Error{
						Code:        errCodes.CantSave,
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"username": &Error{
						Code:        errCodes.FailedToValidate,
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"password": &Error{
						Code:        errCodes.FailedToValidate,
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"password": &Error{
						Code:        errCodes.FailedToValidate,
						Description: `Password is empty`,
					},
					"username": &Error{
						Code:        errCodes.FailedToValidate,
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
	resp   *FormErrors
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"pattern_mismatch": {
						Code:        errCodes.RowNotFound,
						Description: "Can't find user with given parameters",
					},
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
			resp: &FormErrors{
				Errors: map[string]*Error{
					"pattern_mismatch": {
						Code:        errCodes.RowNotFound,
						Description: "Can't find user with given parameters",
					},
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
