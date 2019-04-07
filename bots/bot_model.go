package bots

import (
	"crypto/sha1"

	"github.com/go-park-mail-ru/2019_1_HotCode/database"
	"github.com/go-park-mail-ru/2019_1_HotCode/games"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/jackc/pgx"

	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
)

// BotAccessObject DAO for Bot model
type BotAccessObject interface {
	Create(b *BotModel) error
	SetBotVerifiedByID(botID int64, isActive bool) error
	GetBotsByAuthorID(authorID int64) ([]*BotModel, error)
	GetBotsByGameSlugAndAuthorID(authorID int64, slug string) ([]*BotModel, error)
}

// AccessObject implementation of BotAccessObject
type AccessObject struct{}

var Bots BotAccessObject

func init() {
	Bots = &AccessObject{}
}

// Bot mode for bots table
type BotModel struct {
	ID         pgtype.Int8
	Code       pgtype.Text
	Language   pgtype.Varchar
	IsActive   pgtype.Bool
	IsVerified pgtype.Bool
	AuthorID   pgtype.Int8
	GameSlug   pgtype.Varchar

	codeHash pgtype.Bytea
}

func (bd *AccessObject) Create(b *BotModel) error {
	b.codeHash = pgtype.Bytea{
		// нам не важна безопасность, только для быстрого поиска дубликатов
		Bytes:  sha1.New().Sum([]byte(b.Code.String)),
		Status: pgtype.Present,
	}

	tx, err := database.Conn.Begin()
	if err != nil {
		return errors.Wrap(err, "can not open bot create transaction")
	}
	defer tx.Rollback()

	g, err := games.Games.GetGameBySlug(b.GameSlug.String)
	if err != nil {
		return errors.Wrap(utils.ErrNotExists, errors.Wrap(err, "can not get game with this slug").Error())
	}

	row := tx.QueryRow(`INSERT INTO bots (code, code_hash, language, author_id, game_id)
	 	VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		&b.Code, &b.codeHash, &b.Language, &b.AuthorID, &g.ID)
	if err = row.Scan(&b.ID); err != nil {
		pgErr, ok := err.(pgx.PgError)
		if !ok {
			return errors.Wrap(err, "can not insert bot row")
		}
		if pgErr.Code == "23505" {
			return errors.Wrap(utils.ErrTaken, errors.Wrap(err, "code duplication").Error())
		}
		return errors.Wrap(pgErr, "can not insert bot row")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "can not commit bot create transaction")
	}

	return nil
}

func (bd *AccessObject) SetBotVerifiedByID(botID int64, isVerified bool) error {
	row := database.Conn.QueryRow(`UPDATE bots SET is_verified = $1 
									WHERE bots.id = $2 RETURNING bots.id;`, isVerified, botID)

	var id int64
	if err := row.Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			return errors.Wrap(utils.ErrNotExists, errors.Wrap(err, "now row to update").Error())
		}

		return errors.Wrap(err, "can not update bot row")
	}

	return nil
}

func (bd *AccessObject) GetBotsByAuthorID(authorID int64) ([]*BotModel, error) {
	return bd.getBotsByGameSlugAndAuthorID(authorID, "")
}

func (bd *AccessObject) GetBotsByGameSlugAndAuthorID(authorID int64, slug string) ([]*BotModel, error) {
	return bd.getBotsByGameSlugAndAuthorID(authorID, slug)
}

func (bd *AccessObject) getBotsByGameSlugAndAuthorID(authorID int64, slug string) ([]*BotModel, error) {
	args := []interface{}{authorID}
	query := `SELECT b.id, b.code, b.language,
	b.is_active, b.is_verified, b.author_id, g.slug 
	FROM bots b LEFT JOIN games g on b.game_id = g.id WHERE b.author_id = $1`
	if slug != "" {
		query += ` AND g.slug = $2`
		args = append(args, slug)
	}
	query += ";"

	rows, err := database.Conn.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "get bots by game slug and author id error")
	}
	defer rows.Close()

	bots := make([]*BotModel, 0)
	for rows.Next() {
		bot := &BotModel{}
		err = rows.Scan(&bot.ID, &bot.Code,
			&bot.Language, &bot.IsActive, &bot.IsVerified,
			&bot.AuthorID, &bot.GameSlug)
		if err != nil {
			return nil, errors.Wrap(err, "get bots by game slug and author id scan bot error")
		}
		bots = append(bots, bot)
	}

	return bots, nil
}
