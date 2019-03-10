package models

import (
	"database/sql"

	"github.com/pkg/errors"
)

// Game модель для таблицы games
type Game struct {
	ID    int64
	Title string
}

type ScoredUser struct {
	User
	Score int
}

// GetGameByID получаем инфу по игре по её ID
func GetGameByID(id int64) (*Game, error) {
	g := &Game{}
	// начинаем переезд с GORM(уже созданые методы будут обновлены позже)
	row := db.DB().QueryRow(`SELECT * FROM games WHERE id = $1`, id)
	if err := row.Scan(&g.ID, &g.Title); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotExists
		}

		return nil, errors.Wrap(err, "get game error")
	}

	return g, nil
}

// GetGameLeaderboardByID получаем leaderboard по ID
func GetGameLeaderboardByID(id int64, limit, offset int) ([]*ScoredUser, error) {
	rows, err := db.DB().Query(`SELECT u.id, u.username, u.photo_uuid, u.active, ug.score FROM users u
					LEFT JOIN users_games ug on u.id = ug.user_id
					WHERE ug.game_id = $1 ORDER BY ug.score DESC OFFSET $2 LIMIT $3;`, id, offset, limit)
	if err != nil {
		return nil, errors.Wrap(err, "get leaderboard error")
	}

	leaderboard := make([]*ScoredUser, 0)
	for rows.Next() {
		scoredUser := &ScoredUser{}
		err = rows.Scan(&scoredUser.ID, &scoredUser.Username,
			&scoredUser.PhotoUUID, &scoredUser.Active,
			&scoredUser.Score)
		if err != nil {
			return nil, errors.Wrap(err, "get leaderboard scan user error")
		}
		leaderboard = append(leaderboard, scoredUser)
	}

	return leaderboard, nil
}
