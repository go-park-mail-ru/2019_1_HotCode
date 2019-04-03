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
}

// AccessObject implementation of BotAccessObject
type AccessObject struct{}

var Bots BotAccessObject

func init() {
	Bots = &AccessObject{}
}

// Bot mode for bots table
type BotModel struct {
	ID       pgtype.Int8
	Code     pgtype.Text
	Language pgtype.Varchar
	IsActive pgtype.Bool
	AuthorID pgtype.Int8
	GameSlug pgtype.Varchar

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
