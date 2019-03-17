package controllers

import (
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/go-park-mail-ru/2019_1_HotCode/models"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// GetGame получает объект игры
func GetGame(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "GetGame")
	errWriter := NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	gameID, err := strconv.ParseInt(vars["game_id"], 10, 64)
	if err != nil {
		errWriter.WriteError(http.StatusNotFound, errors.Wrap(err, "wrong format game_id"))
		return
	}

	game, err := models.Games.GetGameByID(gameID)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "game not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get game method error"))
		}
		return
	}

	utils.WriteApplicationJSON(w, http.StatusOK, &Game{
		ID:    game.ID.Int,
		Title: game.Title.String,
	})
}

// GetGameList gets list of games
func GetGameList(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "GetGameList")
	errWriter := NewErrorResponseWriter(w, logger)

	games, err := models.Games.GetGameList()
	if err != nil {
		errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get game list method error"))

		return
	}

	respGames := make([]*Game, len(games))
	for i, game := range games {
		respGames[i] = &Game{
			ID:    game.ID.Int,
			Title: game.Title.String,
		}
	}

	utils.WriteApplicationJSON(w, http.StatusOK, respGames)
}

// GetGameLeaderboard gets list of leaders in game
func GetGameLeaderboard(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "GetGameLeaderboard")
	errWriter := NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	gameID, err := strconv.ParseInt(vars["game_id"], 10, 64)
	if err != nil {
		errWriter.WriteError(http.StatusNotFound, errors.Wrap(err, "wrong format game_id"))
		return
	}

	query := r.URL.Query()
	limitParam, err := strconv.Atoi(query.Get("limit"))
	if err != nil {
		limitParam = 5
	}
	offsetParam, err := strconv.Atoi(query.Get("offset"))
	if err != nil {
		offsetParam = 0
	}

	leadersModels, err := models.Games.GetGameLeaderboardByID(gameID, limitParam, offsetParam)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "game not exists or offset is large"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get game method error"))
		}
		return
	}

	leaders := make([]*ScoredUser, len(leadersModels))
	for i, leader := range leadersModels {
		leaders[i] = &ScoredUser{
			InfoUser: InfoUser{
				BasicUser: BasicUser{
					Username: leader.Username.String,
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
	logger := getLogger(r, "GetGameTotalPlayers")
	errWriter := NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	gameID, err := strconv.ParseInt(vars["game_id"], 10, 64)
	if err != nil {
		errWriter.WriteError(http.StatusNotFound, errors.Wrap(err, "wrong format game_id"))
		return
	}

	totalCount, err := models.Games.GetGameTotalPlayersByID(gameID)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
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