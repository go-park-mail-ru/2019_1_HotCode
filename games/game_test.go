package games

import (
	"testing"

	"github.com/go-park-mail-ru/2019_1_HotCode/testutils"
	"github.com/go-park-mail-ru/2019_1_HotCode/users"
	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/jackc/pgx/pgtype"

	log "github.com/sirupsen/logrus"
)

func init() {
	// чтобы не заваливать всё логами
	log.SetLevel(log.PanicLevel)
}

type GameTest struct {
	games    map[string]*GameModel
	nextFail error
}

// setFailureUser fails next request
func setFailureGame(err error) {
	bt := Games.(*GameTest)
	bt.nextFail = err
}

func (gt *GameTest) GetGameBySlug(slug string) (*GameModel, error) {
	if gt.nextFail != nil {
		err := gt.nextFail
		gt.nextFail = nil
		return nil, err
	}

	g := gt.games[slug]
	return g, nil
}

func (gt *GameTest) GetGameTotalPlayersBySlug(slug string) (int64, error) {
	if gt.nextFail != nil {
		err := gt.nextFail
		gt.nextFail = nil
		return 0, err
	}

	return 1, nil
}

func (gt *GameTest) GetGameList() ([]*GameModel, error) {
	if gt.nextFail != nil {
		err := gt.nextFail
		gt.nextFail = nil
		return nil, err
	}

	games := make([]*GameModel, 0, len(gt.games))
	for _, game := range gt.games {
		games = append(games, game)
	}

	return games, nil
}

func (gt *GameTest) GetGameLeaderboardBySlug(slug string, limit, offset int) ([]*ScoredUserModel, error) {
	if gt.nextFail != nil {
		err := gt.nextFail
		gt.nextFail = nil
		return nil, err
	}

	leaderboard := []*ScoredUserModel{
		{
			UserModel: users.UserModel{
				ID:       pgtype.Int8{Int: 1, Status: pgtype.Present},
				Username: pgtype.Varchar{String: "GDVFox", Status: pgtype.Present},
				PhotoUUID: pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
					Status: pgtype.Present},
			},
			Score: pgtype.Int4{Int: 1337, Status: pgtype.Present},
		},
	}

	return leaderboard, nil
}

func initTests() {
	Games = &GameTest{
		games: map[string]*GameModel{
			"pong": {
				ID:          pgtype.Int8{Int: 1, Status: pgtype.Present},
				Slug:        pgtype.Text{String: "pong", Status: pgtype.Present},
				Title:       pgtype.Text{String: "Pong", Status: pgtype.Present},
				Description: pgtype.Text{String: "Very cool game(net)", Status: pgtype.Present},
				Rules:       pgtype.Text{String: "Do not cheat, please", Status: pgtype.Present},
				CodeExample: pgtype.Text{String: "const a = 5;", Status: pgtype.Present},
				BotCode:     pgtype.Text{String: "const a = 5;", Status: pgtype.Present},
				LogoUUID: pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
					Status: pgtype.Present},
				BackgroundUUID: pgtype.UUID{Bytes: [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
					Status: pgtype.Present},
			},
		},
		nextFail: nil,
	}
}

type GameTestCase struct {
	testutils.Case
	Failure error
}

func runTableAPITests(t *testing.T, cases []*GameTestCase) {
	for i, c := range cases {
		runAPITest(t, i, c)
	}
}

func runAPITest(t *testing.T, i int, c *GameTestCase) {
	if c.Failure != nil {
		setFailureGame(c.Failure)
	}

	testutils.RunAPITest(t, i, &c.Case)
}

func TestGetGame(t *testing.T) {
	initTests()

	cases := []*GameTestCase{
		{ // Всё ок
			Case: testutils.Case{
				ExpectedCode: 200,
				ExpectedBody: `{"slug":"pong","title":"Pong","background_uuid":"00010203-0405-0607-0809-0a0b0c0d0e0f",` +
					`"description":"Very cool game(net)","rules":"Do not cheat, please",` +
					`"code_example":"const a = 5;","bot_code":"const a = 5;",` +
					`"logo_uuid":"01020304-0506-0708-090a-0b0c0d0e0f10"}`,
				Method:   "GET",
				Pattern:  "/games/{game_slug}",
				Endpoint: "/games/pong",
				Function: GetGame,
			},
		},
		{ // Такой игрули нет
			Case: testutils.Case{
				ExpectedCode: 404,
				ExpectedBody: `{"message":"game not exists: not_exists"}`,
				Method:       "GET",
				Pattern:      "/games/{game_slug}",
				Endpoint:     "/games/not_pong",
				Function:     GetGame,
			},
			Failure: utils.ErrNotExists,
		},
		{ // база сломалась
			Case: testutils.Case{
				ExpectedCode: 500,
				ExpectedBody: `{"message":"get game method error: internal server error"}`,
				Method:       "GET",
				Pattern:      "/games/{game_slug}",
				Endpoint:     "/games/not_pong",
				Function:     GetGame,
			},
			Failure: utils.ErrInternal,
		},
	}

	runTableAPITests(t, cases)
}

func TestGetGameList(t *testing.T) {
	initTests()

	cases := []*GameTestCase{
		{ // Всё ок
			Case: testutils.Case{
				ExpectedCode: 200,
				ExpectedBody: `[{"slug":"pong","title":"Pong","background_uuid":"00010203-0405-0607-0809-0a0b0c0d0e0f"}]`,
				Method:       "GET",
				Pattern:      "/games",
				Endpoint:     "/games",
				Function:     GetGameList,
			},
		},
		{ // база сломалась
			Case: testutils.Case{
				ExpectedCode: 500,
				ExpectedBody: `{"message":"get game list method error: internal server error"}`,
				Method:       "GET",
				Pattern:      "/games",
				Endpoint:     "/games",
				Function:     GetGameList,
			},
			Failure: utils.ErrInternal,
		},
	}

	runTableAPITests(t, cases)
}

func TestGetGameLeaderboard(t *testing.T) {
	initTests()

	cases := []*GameTestCase{
		{ // Всё ок
			Case: testutils.Case{
				ExpectedCode: 200,
				ExpectedBody: `[{"username":"GDVFox","photo_uuid":"01020304-0506-0708-090a-0b0c0d0e0f10","id":1,` +
					`"active":false,"score":1337}]`,
				Method:   "GET",
				Pattern:  "/games/{game_slug}/leaderboard",
				Endpoint: "/games/pong/leaderboard",
				Function: GetGameLeaderboard,
			},
		},
		{ // Такой игрули нет
			Case: testutils.Case{
				ExpectedCode: 404,
				ExpectedBody: `{"message":"game not exists or offset is large: not_exists"}`,
				Method:       "GET",
				Pattern:      "/games/{game_slug}/leaderboard",
				Endpoint:     "/games/pong/leaderboard",
				Function:     GetGameLeaderboard,
			},
			Failure: utils.ErrNotExists,
		},
		{ // база сломалась
			Case: testutils.Case{
				ExpectedCode: 500,
				ExpectedBody: `{"message":"get game method error: internal server error"}`,
				Method:       "GET",
				Pattern:      "/games/{game_slug}/leaderboard",
				Endpoint:     "/games/pong/leaderboard",
				Function:     GetGameLeaderboard,
			},
			Failure: utils.ErrInternal,
		},
	}

	runTableAPITests(t, cases)
}

func TestGetGameTotalPlayers(t *testing.T) {
	initTests()

	cases := []*GameTestCase{
		{ // Всё ок
			Case: testutils.Case{
				ExpectedCode: 200,
				ExpectedBody: `{"count":1}`,
				Method:       "GET",
				Pattern:      "/games/{game_slug}/leaderboard/count",
				Endpoint:     "/games/pong/leaderboard/count",
				Function:     GetGameTotalPlayers,
			},
		},
		{ // Такой игрули нет
			Case: testutils.Case{
				ExpectedCode: 404,
				ExpectedBody: `{"message":"game not exists: not_exists"}`,
				Method:       "GET",
				Pattern:      "/games/{game_slug}/leaderboard/count",
				Endpoint:     "/games/pong/leaderboard/count",
				Function:     GetGameTotalPlayers,
			},
			Failure: utils.ErrNotExists,
		},
		{ // база сломалась
			Case: testutils.Case{
				ExpectedCode: 500,
				ExpectedBody: `{"message":"get game method error: internal server error"}`,
				Method:       "GET",
				Pattern:      "/games/{game_slug}/leaderboard/count",
				Endpoint:     "/games/pong/leaderboard/count",
				Function:     GetGameTotalPlayers,
			},
			Failure: utils.ErrInternal,
		},
	}

	runTableAPITests(t, cases)
}
