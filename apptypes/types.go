package apptypes

// ContextKey ключ для контекста реквеста
type ContextKey int

const (
	// UserInfoKey ключ, по которому в контексте
	// реквеста хранится структура юзера после валидации
	UserInfoKey ContextKey = 1
)

// BasicUser базовые поля
type BasicUser struct {
	Username string `json:"username"`
}

// InfoUser BasicUser, расширенный служебной инфой
type InfoUser struct {
	BasicUser
	ID     int64 `json:"id"`
	Active bool  `json:"active"`
}

// FormUser BasicUser, расширенный паролем, используется для входа и регистрации
type FormUser struct {
	BasicUser
	Password string `json:"password"`
}
