package bots

import (
	"encoding/json"

	"github.com/go-park-mail-ru/2019_1_HotCode/queue"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"

	log "github.com/sirupsen/logrus"
)

const (
	testerQueueName = "tester_rpc_queue"
)

type TesterStatusQueue struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

type TesterStatusUpdate struct {
	NewStatus string `json:"new_status"`
}

type TesterStatusResult struct {
	IsVerified bool   `json:"verified"`
	Message    string `json:"message"`
}

func sendForVerifyRPC(bot *BotUpload) (<-chan *TesterStatusQueue, error) {
	respQ, err := queue.Channel.QueueDeclare(
		"", // пакет amqp сам сгенерит
		false,
		true,
		false, // удаляем после того, как процедура отработала
		false,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "can not create queue for responses")
	}

	requestUUID := uuid.New().String()
	resps, err := queue.Channel.Consume(
		respQ.Name,
		requestUUID,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "can not register a consumer")
	}

	body, err := json.Marshal(bot)
	if err != nil {
		return nil, errors.Wrap(err, "can not marshal bot info")
	}

	err = queue.Channel.Publish(
		"",
		testerQueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: requestUUID,
			ReplyTo:       respQ.Name,
			Body:          body,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "can not publish a message")
	}

	events := make(chan *TesterStatusQueue)
	go func(in <-chan amqp.Delivery, out chan<- *TesterStatusQueue, corrID string) {
		for resp := range in {
			if corrID != resp.CorrelationId {
				continue
			}

			testerResp := &TesterStatusQueue{}
			err := json.Unmarshal(resp.Body, testerResp)
			if err != nil {
				log.WithField("method", "sendForVerifyRPC goroutine").Error(errors.Wrap(err, "unmarshal tester response error"))
				break
			}
			out <- testerResp

			if testerResp.Type == "result" {
				// отцепились от очереди -- она удалилась
				err = queue.Channel.Cancel(
					corrID,
					false,
				)
				if err != nil {
					log.WithField("method", "sendForVerifyRPC goroutine").Error(errors.Wrap(err, "queue cancel error"))
				}
			}
		}

		close(out)

	}(resps, events, requestUUID)

	return events, nil
}

func processTestingStatus(botID, authorID int64, gameSlug string,
	broadcast chan<- *BotVerifyStatusMessage, events <-chan *TesterStatusQueue) {

	logger := log.WithFields(log.Fields{
		"bot_id": botID,
		"method": "processTestingStatus",
	})

	status := ""
	for event := range events {
		logger.Infof("Processing [%s]", event.Type)
		switch event.Type {
		case "status":
			upd := &TesterStatusUpdate{}
			err := json.Unmarshal(event.Body, upd)
			if err != nil {
				logger.Error(errors.Wrap(err, "can not unmarshal update status body"))
				continue
			}

			broadcast <- &BotVerifyStatusMessage{
				BotID:     botID,
				AuthorID:  authorID,
				GameSlug:  gameSlug,
				NewStatus: upd.NewStatus,
			}

			status = upd.NewStatus
		case "result":
			res := &TesterStatusResult{}
			err := json.Unmarshal(event.Body, res)
			if err != nil {
				logger.Error(errors.Wrap(err, "can not unmarshal result status body"))
				continue
			}

			newStatus := "Not Verifyed\n"
			if res.IsVerified {
				newStatus = "Verifyed\n"
			}

			newStatus += res.Message
			broadcast <- &BotVerifyStatusMessage{
				BotID:     botID,
				AuthorID:  authorID,
				GameSlug:  gameSlug,
				NewStatus: newStatus,
			}

			err = Bots.SetBotVerifiedByID(botID, res.IsVerified)
			if err != nil {
				logger.Error(errors.Wrap(err, "can update bot active status"))
				continue
			}

			status = newStatus
		default:
			logger.Error(errors.New("can not process unknown status type"))
		}

		logger.Infof("Processing [%s]: new status: %s", event.Type, status)
	}
}
