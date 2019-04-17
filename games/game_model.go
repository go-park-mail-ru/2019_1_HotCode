package games

import (
	"github.com/go-park-mail-ru/2019_1_HotCode/database"
	"github.com/go-park-mail-ru/2019_1_HotCode/users"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"

	// драйвер Database
	"github.com/jackc/pgx"
)

// GameAccessObject DAO for User model
type GameAccessObject interface {
	GetGameBySlug(slug string) (*GameModel, error)
	GetGameTotalPlayersBySlug(slug string) (int64, error)
	GetGameList() ([]*GameModel, error)
	GetGameLeaderboardBySlug(slug string, limit, offset int) ([]*ScoredUserModel, error)
}

// AccessObject implementation of GameAccessObject
type AccessObject struct{}

var Games GameAccessObject

func init() {
	Games = &AccessObject{}
}

// Game модель для таблицы games
type GameModel struct {
	ID             pgtype.Int8
	Slug           pgtype.Text
	Title          pgtype.Text
	Description    pgtype.Text
	Rules          pgtype.Text
	CodeExample    pgtype.Text
	BotCode        pgtype.Text
	LogoUUID       pgtype.UUID
	BackgroundUUID pgtype.UUID
}

// ScoredUser User with score
type ScoredUserModel struct {
	users.UserModel
	Score pgtype.Int4
}

func (gs *AccessObject) GetGameBySlug(slug string) (*GameModel, error) {
	g, err := gs.getGameImpl(database.Conn, "slug", slug)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, utils.ErrNotExists
		}

		return nil, errors.Wrap(err, "get game by slug error")
	}

	return g, nil
}

// GetGameTotalPlayersByID получение общего количества игроков
func (gs *AccessObject) GetGameTotalPlayersBySlug(slug string) (int64, error) {
	tx, err := database.Conn.Begin()
	if err != nil {
		return 0, errors.Wrap(err, "can not open 'GetGameTotalPlayersByID' transaction")
	}
	defer tx.Rollback()

	g, err := gs.getGameImpl(tx, "slug", slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, utils.ErrNotExists
		}

		return 0, errors.Wrap(err, "'GetGameTotalPlayersByID' can not get game by id")
	}

	var totalPlayers int64
	row := tx.QueryRow(`SELECT count(*) FROM users_games WHERE game_id = $1;`, &g.ID)
	if err = row.Scan(&totalPlayers); err != nil {
		return 0, errors.Wrap(err, "get game total players error")
	}

	err = tx.Commit()
	if err != nil {
		return 0, errors.Wrap(err, "'GetGameTotalPlayersByID' transaction commit error")
	}

	return totalPlayers, nil
}

// GetGameLeaderboardBySlug получаем leaderboard по slug
func (gs *AccessObject) GetGameLeaderboardBySlug(slug string, limit, offset int) ([]*ScoredUserModel, error) {
	// узнаём количество

	rows, err := database.Conn.Query(`SELECT u.id, u.username, u.photo_uuid, u.active, ug.score FROM users u
					LEFT JOIN users_games ug on u.id = ug.user_id
					RIGHT JOIN games g on ug.game_id = g.id
					WHERE g.slug = $1 ORDER BY ug.score DESC OFFSET $2 LIMIT $3;`, slug, offset, limit)
	if err != nil {
		return nil, errors.Wrap(err, "get leaderboard error")
	}
	defer rows.Close()

	leaderboard := make([]*ScoredUserModel, 0)
	for rows.Next() {
		scoredUser := &ScoredUserModel{}
		err = rows.Scan(&scoredUser.ID, &scoredUser.Username,
			&scoredUser.PhotoUUID, &scoredUser.Active,
			&scoredUser.Score)
		if err != nil {
			return nil, errors.Wrap(err, "get leaderboard scan user error")
		}
		leaderboard = append(leaderboard, scoredUser)
	}

	if len(leaderboard) == 0 {
		return nil, utils.ErrNotExists
	}

	return leaderboard, nil
}

// GetGameList returns full list of active games
func (gs *AccessObject) GetGameList() ([]*GameModel, error) {
	rows, err := database.Conn.Query(`SELECT g.id, g.slug, g.title, g.description,
								g.rules, g.code_example, g.bot_code, g.logo_uuid, g.background_uuid
								FROM games g`)
	if err != nil {
		return nil, errors.Wrap(err, "get game list error")
	}
	defer rows.Close()

	games := make([]*GameModel, 0)
	for rows.Next() {
		g := &GameModel{}
		err = rows.Scan(&g.ID, &g.Slug, &g.Title, &g.Description,
			&g.Rules, &g.CodeExample, &g.BotCode, &g.LogoUUID, &g.BackgroundUUID)
		if err != nil {
			return nil, errors.Wrap(err, "get games scan game error")
		}
		games = append(games, g)
	}

	return games, nil
}

func (gs *AccessObject) getGameImpl(q database.Queryer, field, value string) (*GameModel, error) {
	g := &GameModel{}

	row := q.QueryRow(`SELECT g.id, g.slug, g.title, g.description,
						g.rules, g.code_example, g.bot_code, g.logo_uuid, g.background_uuid
						FROM games g WHERE `+field+` = $1;`, value)
	if err := row.Scan(&g.ID, &g.Slug, &g.Title, &g.Description,
		&g.Rules, &g.CodeExample, &g.BotCode, &g.LogoUUID, &g.BackgroundUUID); err != nil {
		return nil, err
	}

	return g, nil
}
