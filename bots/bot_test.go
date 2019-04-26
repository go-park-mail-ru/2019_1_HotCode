package bots

import (
	"testing"

	"github.com/go-park-mail-ru/2019_1_HotCode/games"

	"github.com/go-park-mail-ru/2019_1_HotCode/testutils"

	"github.com/jackc/pgx/pgtype"
	log "github.com/sirupsen/logrus"
)

func init() {
	// чтобы не заваливать всё логами
	log.SetLevel(log.PanicLevel)
}

type BotTest struct {
	ids      int64
	bots     map[int64]BotModel
	nextFail error
}

func (bt *BotTest) newID() int64 {
	bt.ids++
	return bt.ids - 1
}

// setFailureUser fails next request
func setFailureBot(err error) {
	bt := Bots.(*BotTest)
	bt.nextFail = err
}

func (bt *BotTest) Create(b *BotModel) error {
	if bt.nextFail != nil {
		err := bt.nextFail
		bt.nextFail = nil
		return err
	}

	b.IsActive = pgtype.Bool{Bool: false, Status: pgtype.Present}
	b.ID = pgtype.Int8{Int: bt.newID(), Status: pgtype.Present}
	bt.bots[b.ID.Int] = *b
	return nil
}

func (bt *BotTest) SetBotVerifiedByID(botID int64, isActive bool) error {
	return nil
}

func (bt *BotTest) GetBotsByAuthorID(authorID int64) ([]*BotModel, error) {
	return nil, nil
}

func (bt *BotTest) GetBotsByGameSlugAndAuthorID(authorID int64, slug string) ([]*BotModel, error) {
	return nil, nil
}

type GameTest struct {
	games    map[string]games.GameModel
	nextFail error
}

func (gt *GameTest) GetGameBySlug(slug string) (*games.GameModel, error) {
	g := gt.games[slug]
	return &g, nil
}

func (gt *GameTest) GetGameTotalPlayersBySlug(slug string) (int64, error) {
	return 0, nil
}

func (gt *GameTest) GetGameList() ([]*games.GameModel, error) {
	return nil, nil
}

func (gt *GameTest) GetGameLeaderboardBySlug(slug string, limit, offset int) ([]*games.ScoredUserModel, error) {
	return nil, nil
}

func initTests() {
	Bots = &BotTest{
		ids:      1,
		bots:     make(map[int64]BotModel),
		nextFail: nil,
	}

	games.Games = &GameTest{
		games: map[string]games.GameModel{
			"pong": {
				ID:   pgtype.Int8{Int: 1, Status: pgtype.Present},
				Slug: pgtype.Text{String: "pong", Status: pgtype.Present},
			},
		},
		nextFail: nil,
	}
}

type BotTestCase struct {
	testutils.Case
	Failure error
}

func runTableAPITests(t *testing.T, cases []*BotTestCase) {
	for i, c := range cases {
		runAPITest(t, i, c)
	}
}

func runAPITest(t *testing.T, i int, c *BotTestCase) {
	if c.Failure != nil {
		setFailureBot(c.Failure)
	}

	testutils.RunAPITest(t, i, &c.Case)
}

// Отключены, так как пока что нет мокапа для RabbitMQ
func TestCreateBot(t *testing.T) {

	//initTests()

	// cases := []*BotTestCase{
	// 	{ // Без токена
	// 		Case: testutils.Case{
	// 			Payload:      []byte(`{"code":"const a=0","game_slug":"pong", "lang":"CPP"}`),
	// 			ExpectedCode: 401,
	// 			ExpectedBody: `{"message":"session info is not presented"}`,
	// 			Method:       "POST",
	// 			Pattern:      "/bots",
	// 			Function:     CreateBot,
	// 		},
	// 	},
	// 	{ // Неподдерживаемый язык
	// 		Case: testutils.Case{
	// 			Payload:      []byte(`{"code":"const a=0","game_slug":"pong", "lang":"CPP"}`),
	// 			ExpectedCode: 400,
	// 			ExpectedBody: `{"lang":"invalid"}`,
	// 			Method:       "POST",
	// 			Pattern:      "/bots",
	// 			Function:     CreateBot,
	// 			Context: context.WithValue(context.Background(),
	// 				users.SessionInfoKey, &users.SessionPayload{ID: 1, PwdVer: 1}),
	// 		},
	// 	},
	// 	{ // Кривой JSON (без запятых)
	// 		Case: testutils.Case{
	// 			Payload:      []byte(`{"code":"const a=0" "game_slug":"pong" "lang":"CPP"}`),
	// 			ExpectedCode: 400,
	// 			ExpectedBody: `{"message":"decode body error: invalid character '\"' after object key:value pair"}`,
	// 			Method:       "POST",
	// 			Pattern:      "/bots",
	// 			Function:     CreateBot,
	// 			Context: context.WithValue(context.Background(),
	// 				users.SessionInfoKey, &users.SessionPayload{ID: 1, PwdVer: 1}),
	// 		},
	// 	},
	// 	{ // Создали бота
	// 		Case: testutils.Case{
	// 			Payload:      []byte(`{"code":"const a=0","game_slug":"pong", "lang":"JS"}`),
	// 			ExpectedCode: 200,
	// 			ExpectedBody: `{"id":1,"game_slug":"pong","author_id":1,"is_active":false,"code":"const a=0","lang":"JS"}`,
	// 			Method:       "POST",
	// 			Pattern:      "/bots",
	// 			Function:     CreateBot,
	// 			Context: context.WithValue(context.Background(),
	// 				users.SessionInfoKey, &users.SessionPayload{ID: 1, PwdVer: 1}),
	// 		},
	// 	},
	// 	{ // Создали дубликат
	// 		Case: testutils.Case{
	// 			Payload:      []byte(`{"code":"const a=0","game_slug":"pong", "lang":"JS"}`),
	// 			ExpectedCode: 400,
	// 			ExpectedBody: `{"code":"taken"}`,
	// 			Method:       "POST",
	// 			Pattern:      "/bots",
	// 			Function:     CreateBot,
	// 			Context: context.WithValue(context.Background(),
	// 				users.SessionInfoKey, &users.SessionPayload{ID: 1, PwdVer: 1}),
	// 		},
	// 		Failure: utils.ErrTaken,
	// 	},
	// 	{ // Сломалась база
	// 		Case: testutils.Case{
	// 			Payload:      []byte(`{"code":"const a=0","game_slug":"pong", "lang":"JS"}`),
	// 			ExpectedCode: 500,
	// 			ExpectedBody: `{"message":"user create error: internal server error"}`,
	// 			Method:       "POST",
	// 			Pattern:      "/bots",
	// 			Function:     CreateBot,
	// 			Context: context.WithValue(context.Background(),
	// 				users.SessionInfoKey, &users.SessionPayload{ID: 1, PwdVer: 1}),
	// 		},
	// 		Failure: utils.ErrInternal,
	// 	},
	// 	{ // Нет такой игры
	// 		Case: testutils.Case{
	// 			Payload:      []byte(`{"code":"const a=0","game_slug":"pong", "lang":"JS"}`),
	// 			ExpectedCode: 400,
	// 			ExpectedBody: `{"game_slug":"not_exists"}`,
	// 			Method:       "POST",
	// 			Pattern:      "/bots",
	// 			Function:     CreateBot,
	// 			Context: context.WithValue(context.Background(),
	// 				users.SessionInfoKey, &users.SessionPayload{ID: 1, PwdVer: 1}),
	// 		},
	// 		Failure: utils.ErrNotExists,
	// 	},
	// }

	//runTableAPITests(t, cases)
}
