package games

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"

	"github.com/go-park-mail-ru/2019_1_HotCode/users"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// GetGame получает объект игры
func GetGame(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "GetGame")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	game, err := Games.GetGameBySlug(vars["game_slug"])
	if err != nil {
		if errors.Cause(err) == utils.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "game not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get game method error"))
		}
		return
	}

	utils.WriteApplicationJSON(w, http.StatusOK, &GameFull{
		Game: Game{
			Slug:           game.Slug.String,
			Title:          game.Title.String,
			BackgroundUUID: uuid.UUID(game.BackgroundUUID.Bytes).String(), // точно 16 байт
		},
		Description: game.Description.String,
		Rules:       game.Rules.String,
		CodeExample: game.CodeExample.String,
		BotCode:     game.BotCode.String,
		LogoUUID:    uuid.UUID(game.LogoUUID.Bytes).String(), // точно 16 байт
	})
}

// GetGameList gets list of games
func GetGameList(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "GetGameList")
	errWriter := utils.NewErrorResponseWriter(w, logger)

	games, err := Games.GetGameList()
	if err != nil {
		errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get game list method error"))

		return
	}

	respGames := make([]*Game, len(games))
	for i, game := range games {
		respGames[i] = &Game{
			Slug:           game.Slug.String,
			Title:          game.Title.String,
			BackgroundUUID: uuid.UUID(game.BackgroundUUID.Bytes).String(), // точно 16 байт
		}
	}

	utils.WriteApplicationJSON(w, http.StatusOK, respGames)
}

// GetGameLeaderboard gets list of leaders in game
func GetGameLeaderboard(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "GetGameLeaderboard")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	query := r.URL.Query()
	limitParam, err := strconv.Atoi(query.Get("limit"))
	if err != nil {
		limitParam = 5
	}
	offsetParam, err := strconv.Atoi(query.Get("offset"))
	if err != nil {
		offsetParam = 0
	}

	leadersModels, err := Games.GetGameLeaderboardBySlug(vars["game_slug"], limitParam, offsetParam)
	if err != nil {
		if errors.Cause(err) == utils.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "game not exists or offset is large"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get game method error"))
		}
		return
	}

	leaders := make([]*ScoredUser, len(leadersModels))
	for i, leader := range leadersModels {
		photoUUID := ""
		if leader.PhotoUUID.Status == pgtype.Present {
			photoUUID = uuid.UUID(leader.PhotoUUID.Bytes).String()
		}

		leaders[i] = &ScoredUser{
			InfoUser: users.InfoUser{
				BasicUser: users.BasicUser{
					Username:  leader.Username.String,
					PhotoUUID: photoUUID,
				},
				ID:     leader.ID.Int,
				Active: leader.Active.Bool,
			},
			Score: leader.Score.Int,
		}
	}

	utils.WriteApplicationJSON(w, http.StatusOK, leaders)
}

// GetGameTotalPlayers количество юзеров игравших в game_id
func GetGameTotalPlayers(w http.ResponseWriter, r *http.Request) {
	logger := utils.GetLogger(r, "GetGameTotalPlayers")
	errWriter := utils.NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	totalCount, err := Games.GetGameTotalPlayersBySlug(vars["game_slug"])
	if err != nil {
		if errors.Cause(err) == utils.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "game not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get game method error"))
		}
		return
	}

	utils.WriteApplicationJSON(w, http.StatusOK, &struct {
		Count int64 `json:"count"`
	}{
		Count: totalCount,
	})
}
