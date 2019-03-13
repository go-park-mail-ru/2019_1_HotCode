package models

import (
	"strconv"

	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"

	// драйвер Database
	"github.com/jackc/pgx"
)

// GameAccessObject DAO for User model
type GameAccessObject interface {
	GetGameByID(id int64) (*Game, error)
	GetGameTotalPlayersByID(id int64) (int64, error)
	GetGameList() ([]*Game, error)
	GetGameLeaderboardByID(id int64, limit, offset int) ([]*ScoredUser, error)
}

// GamesDB implementation of GameAccessObject
type GamesDB struct{}

// Game модель для таблицы games
type Game struct {
	ID    pgtype.Int8
	Title pgtype.Varchar
}

// TableName возвращает имя таблицы для модели games
func (g *Game) TableName() string {
	return "games"
}

// ScoredUser User with score
type ScoredUser struct {
	User
	Score pgtype.Int4
}

// GetGameByID получаем инфу по игре по её ID
func (gs *GamesDB) GetGameByID(id int64) (*Game, error) {
	g, err := gs.getGameImpl(db.conn, "id", strconv.FormatInt(id, 10))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotExists
		}

		return nil, errors.Wrap(err, "get user by username error")
	}

	return g, nil
}

// GetGameTotalPlayersByID получение общего количества игроков
func (gs *GamesDB) GetGameTotalPlayersByID(id int64) (int64, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, errors.Wrap(err, "can not open 'GetGameTotalPlayersByID' transaction")
	}
	defer tx.Rollback()

	_, err = gs.getGameImpl(tx, "id", strconv.FormatInt(id, 10))
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, ErrNotExists
		}

		return 0, errors.Wrap(err, "'GetGameTotalPlayersByID' can not get game by id")
	}

	var totalPlayers int64
	row := tx.QueryRow(`SELECT count(*) FROM users_games WHERE game_id = $1;`, id)
	if err := row.Scan(&totalPlayers); err != nil {
		return 0, errors.Wrap(err, "get game total players error")
	}

	err = tx.Commit()
	if err != nil {
		return 0, errors.Wrap(err, "'GetGameTotalPlayersByID' transaction commit error")
	}

	return totalPlayers, nil
}

// GetGameLeaderboardByID получаем leaderboard по ID
func (gs *GamesDB) GetGameLeaderboardByID(id int64, limit, offset int) ([]*ScoredUser, error) {
	// узнаём количество

	rows, err := db.conn.Query(`SELECT u.id, u.username, u.photo_uuid, u.active, ug.score FROM users u
					LEFT JOIN users_games ug on u.id = ug.user_id
					WHERE ug.game_id = $1 ORDER BY ug.score DESC OFFSET $2 LIMIT $3;`, id, offset, limit)
	if err != nil {
		return nil, errors.Wrap(err, "get leaderboard error")
	}
	defer rows.Close()

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

	if len(leaderboard) == 0 {
		return nil, ErrNotExists
	}

	return leaderboard, nil
}

// GetGameList returns full list of active games
func (gs *GamesDB) GetGameList() ([]*Game, error) {
	rows, err := db.conn.Query(`SELECT g.id, g.title FROM games g`)
	if err != nil {
		return nil, errors.Wrap(err, "get game list error")
	}

	games := make([]*Game, 0)
	for rows.Next() {
		game := &Game{}
		err = rows.Scan(&game.ID, &game.Title)
		if err != nil {
			return nil, errors.Wrap(err, "get games scan game error")
		}
		games = append(games, game)
	}

	return games, nil
}

func (gs *GamesDB) getGameImpl(q queryer, field, value string) (*Game, error) {
	g := &Game{}

	row := q.QueryRow(`SELECT * FROM `+g.TableName()+` WHERE `+field+` = $1;`, value)
	if err := row.Scan(&g.ID, &g.Title); err != nil {
		return nil, err
	}

	return g, nil
}
