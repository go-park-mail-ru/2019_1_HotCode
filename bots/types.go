package bots

import (
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"
)

type Lang string

type BotUpload struct {
	Code     string `json:"code"`
	GameSlug string `json:"game_slug"`
	Language Lang   `json:"lang"`
}

var availableLanguages = map[Lang]struct{}{
	// JS - JavaScript
	"JS": {},
}

func (bu *BotUpload) Validate() error {
	if _, ok := availableLanguages[bu.Language]; !ok {
		return &utils.ValidationError{
			"lang": utils.ErrInvalid.Error(),
		}
	}

	return nil
}

type Bot struct {
	ID       int64  `json:"id"`
	GameSlug string `json:"game_slug"`
	AuthorID int64  `json:"author_id"`
	IsActive bool   `json:"is_active"`
}

type BotFull struct {
	Bot
	Code     string `json:"code"`
	Language Lang   `json:"lang"`
}
