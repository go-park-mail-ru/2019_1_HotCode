package bots

import (
	"testing"

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

func checkFailureBot() error {
	bt := Bots.(*BotTest)
	if bt.nextFail != nil {
		defer func() {
			bt.nextFail = nil
		}()
		return bt.nextFail
	}
	return nil
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

type BotTestCase struct {
	testutils.Case
	Failure error
}

func initTests() {
	Bots = &BotTest{
		ids:      1,
		bots:     make(map[int64]BotModel),
		nextFail: nil,
	}
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

func TestCreateBot(t *testing.T) {
	initTests()

	cases := []*BotTestCase{}

	runTableAPITests(t, cases)
}
