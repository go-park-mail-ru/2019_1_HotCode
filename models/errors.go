package models

import "github.com/pkg/errors"

var (
	// ErrInvalid у поля неправильный формат
	ErrInvalid = errors.New("invalid")
	// ErrRequired поле обязательно, но не было передано
	ErrRequired = errors.New("required")
	// ErrTaken это поле должно быть уникальным и уже используется
	ErrTaken = errors.New("taken")
	// ErrNotExists такой записи нет
	ErrNotExists = errors.New("not_exists")
	// ErrUsernameTaken имя пользователя занято
	ErrUsernameTaken = errors.New("username taken")
	// ErrInternal всё очень плохо
	ErrInternal = errors.New("internal server error")
)
