package games

import "github.com/go-park-mail-ru/2019_1_HotCode/users"

// ScoredUser инфа о юзере расширенная его баллами
type ScoredUser struct {
	users.InfoUser
	Score int32 `json:"score"`
}

// Game схема объекта игры для карусельки
type Game struct {
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	BackgroundUUID string `json:"background_uuid"`
}

type GameFull struct {
	Game
	Description string `json:"description"`
	Rules       string `json:"rules"`
	CodeExample string `json:"code_example"`
	LogoUUID    string `json:"logo_uuid"`
}
