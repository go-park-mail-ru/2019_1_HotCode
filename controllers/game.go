package controllers

import (
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/go-park-mail-ru/2019_1_HotCode/models"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func GetGame(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "GetGame")
	errWriter := NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	gameID, err := strconv.ParseInt(vars["game_id"], 10, 64)
	if err != nil {
		errWriter.WriteError(http.StatusNotFound, errors.Wrap(err, "wrong format user_id"))
		return
	}

	game, err := models.GetGameByID(gameID)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "user not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get user method error"))
		}
		return
	}

	utils.WriteApplicationJSON(w, http.StatusOK, &Game{
		ID:    game.ID,
		Title: game.Title,
	})
}

func GetGameLeaderboard(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r, "GetGameLeaderboard")
	errWriter := NewErrorResponseWriter(w, logger)
	vars := mux.Vars(r)

	gameID, err := strconv.ParseInt(vars["game_id"], 10, 64)
	if err != nil {
		errWriter.WriteError(http.StatusNotFound, errors.Wrap(err, "wrong format user_id"))
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

	leadersModels, err := models.GetGameLeaderboardByID(gameID, limitParam, offsetParam)
	if err != nil {
		if errors.Cause(err) == models.ErrNotExists {
			errWriter.WriteWarn(http.StatusNotFound, errors.Wrap(err, "user not exists"))
		} else {
			errWriter.WriteError(http.StatusInternalServerError, errors.Wrap(err, "get user method error"))
		}
		return
	}

	leaders := make([]*ScoredUser, len(leadersModels))
	for i, leader := range leadersModels {
		leaders[i] = &ScoredUser{
			InfoUser: InfoUser{
				BasicUser: BasicUser{
					Username: leader.Username,
				},
				ID:     leader.ID,
				Active: leader.Active,
			},
			Score: leader.Score,
		}
	}

	utils.WriteApplicationJSON(w, http.StatusOK, leaders)
}
