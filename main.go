package main

import (
	"time"
	"ws-test-task/client"
	"ws-test-task/entity"
	"ws-test-task/server"

	"github.com/rs/zerolog/log"
)

func main() {
	w := server.NewWriter()
	go func() {
		if err := w.Run(); err != nil {
			log.Err(err).Send()
		}
	}()

	wsClient := client.NewClient(20*time.Second, "foo", "bar")
	wsClient.SetSendFunc(w.Send)
	wsClient.SetErrorHandler(errorHandler)
	wsClient.SetResponseHandler(responseHandler)
	wsClient.SetStatusHandler(statusHandler)

	err := wsClient.Auth()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	requestID, err := wsClient.Subscribe("BTC/USDT")
	if err != nil {
		log.Fatal().Err(err).Str("request_id", requestID).Send()
	}
	log.Info().Str("request_id", requestID).Str("symbol", "BTC/USDT").Msg("subscribed to symbol")

	err = wsClient.Listen(w.Read())
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func errorHandler(err error) {
	log.Err(err).Send()
}

func responseHandler(res entity.MethodResponse) {
	if res.Method == entity.MethodExecutions {
		l := log.Info()
		for k, v := range res.Data {
			l.Str(k, v)
		}
		l.Msg("new price update")
	}
}

func statusHandler(status entity.StatusResponse) {
	l := log.Info().Str("request_id", status.ReqID).Bool("ok", status.Status)
	if len(status.Error) > 0 {
		l.Str("error", status.Error)
	}
	l.Msg("status update received")
}
