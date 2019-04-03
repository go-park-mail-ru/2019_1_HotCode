package bots

import (
	"net/http"

	"github.com/go-park-mail-ru/2019_1_HotCode/users"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
)

func CreateBot(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "CreateBot")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	info := users.SessionInfo(r)

	form := &BotUpload{}
	err := utils.DecodeBodyJSON(r.Body, form)
	if err != nil {
		errWriter.WriteWarn(http.StatusBadRequest, errors.Wrap(err, "decode body error"))
		return
	}

	if err = form.Validate(); err != nil {
		// уверены в преобразовании
		errWriter.WriteValidationError(err.(*utils.ValidationError))
		return
	}

	bot := &BotModel{
		Code:     pgtype.Text{String: form.Code, Status: pgtype.Present},
		Language: pgtype.Varchar{String: string(form.Language), Status: pgtype.Present},
		GameSlug: pgtype.Varchar{String: form.GameSlug, Status: pgtype.Present},
		AuthorID: pgtype.Int8{Int: info.ID, Status: pgtype.Present},
	}

	if err = Bots.Create(bot); err != nil {
		switch errors.Cause(err) {
		case utils.ErrNotExists:
			errWriter.WriteValidationError(&utils.ValidationError{
				"game_slug": utils.ErrNotExists.Error(),
			})
		case utils.ErrTaken:
			errWriter.WriteValidationError(&utils.ValidationError{
				"code": utils.ErrTaken.Error(),
			})
		default:
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "user create error"))
		}
		return
	}

	botFull := BotFull{
		Bot: Bot{
			ID:       bot.ID.Int,
			AuthorID: bot.AuthorID.Int,
			IsActive: bot.IsActive.Bool,
			GameSlug: bot.GameSlug.String,
		},
		Code:     form.Code,
		Language: form.Language,
	}

	utils.WriteApplicationJSON(w, http.StatusOK, botFull)
}
