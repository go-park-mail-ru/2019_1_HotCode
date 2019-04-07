package bots

import (
	"net/http"

	"github.com/go-park-mail-ru/2019_1_HotCode/users"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
)

func CreateBot(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "CreateBot")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	info := users.SessionInfo(r)
	if info == nil {
		errWriter.WriteWarn(http.StatusUnauthorized, errors.New("session info is not presented"))
		return
	}

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
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "bot create error"))
		}
		return
	}

	botFull := BotFull{
		Bot: Bot{
			ID:         bot.ID.Int,
			AuthorID:   bot.AuthorID.Int,
			IsActive:   bot.IsActive.Bool,
			IsVerified: bot.IsVerified.Bool,
			GameSlug:   bot.GameSlug.String,
		},
		Code:     form.Code,
		Language: form.Language,
	}

	// делаем RPC запрос
	events, err := sendForVerifyRPC(form)
	if err != nil {
		errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "can not call verify rpc"))
		return
	}

	// запускаем обработчик ответа RPC
	go processTestingStatus(bot.ID.Int, info.ID, bot.GameSlug.String, h.broadcast, events)
	utils.WriteApplicationJSON(w, http.StatusOK, botFull)
}

// GetBotsList TODO: author_id parameter
func GetBotsList(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "GetBotsList")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	info := users.SessionInfo(r)
	if info == nil {
		errWriter.WriteWarn(http.StatusUnauthorized, errors.New("session info is not presented"))
		return
	}

	gameSlug := r.URL.Query().Get("game_slug")
	var err error
	var bots []*BotModel
	if gameSlug == "" {
		bots, err = Bots.GetBotsByAuthorID(info.ID)
	} else {
		bots, err = Bots.GetBotsByGameSlugAndAuthorID(info.ID, gameSlug)
	}
	if err != nil {
		errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get bot method error"))
		return
	}

	respBots := make([]*Bot, len(bots))
	for i, bot := range bots {
		respBots[i] = &Bot{
			ID:         bot.ID.Int,
			GameSlug:   bot.GameSlug.String,
			AuthorID:   bot.AuthorID.Int,
			IsActive:   bot.IsActive.Bool,
			IsVerified: bot.IsVerified.Bool,
		}
	}

	utils.WriteApplicationJSON(w, http.StatusOK, respBots)
}

func OpenVerifyWS(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "GetBotsList")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	info := users.SessionInfo(r)
	if info == nil {
		errWriter.WriteWarn(http.StatusUnauthorized, errors.New("session info is not presented"))
		return
	}

	gameSlug := r.URL.Query().Get("game_slug")
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // мы уже прошли слой CORS
		},
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "upgrade to websocket error"))
		return
	}

	sessionID := uuid.New().String()
	verifyClient := &BotVerifyClient{
		SessionID: sessionID,
		UserID:    info.ID,
		GameSlug:  gameSlug,

		h:    h,
		conn: c,
		send: make(chan *BotVerifyStatusMessage),
	}
	verifyClient.h.register <- verifyClient

	go verifyClient.WriteStatusUpdates()
	go verifyClient.WaitForClose()
}
