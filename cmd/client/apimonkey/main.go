package main

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"meow.tf/streamdeck/sdk"
)

var lg zerolog.Logger
var contextApp string
var globalConfig *config

func main() {
	logFile := &lumberjack.Logger{
		Filename:   "logs/log.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     10,
		Compress:   false,
	}

	lg = zerolog.New(zerolog.MultiLevelWriter(os.Stdout, logFile)).With().Timestamp().Logger()

	sdk.AddHandler(func(event *sdk.WillAppearEvent) {
		contextApp = event.Context
		if event.Payload == nil {
			return
		}

		settingsBytes := event.Payload.Get("settings").MarshalTo(nil)
		lg.Debug().Msgf("Got configuration: %v", string(settingsBytes))

		if err := json.Unmarshal(settingsBytes, &globalConfig); err != nil {
			lg.Err(err).Send()
			sdk.ShowAlert(contextApp)
			return
		}

		lg.Info().Msg("config set")
		go process()
	})

	sdk.AddHandler(func(event *sdk.KeyDownEvent) {
		if globalConfig == nil {
			lg.Error().Msg("global config not set")
			sdk.ShowAlert(contextApp)
			return
		}

		targetUrl := globalConfig.BrowserUrl
		if targetUrl == "" {
			targetUrl = globalConfig.ApiUrl
		}

		if err := exec.Command("rundll32",
			"url.dll,FileProtocolHandler", targetUrl).Start(); err != nil {

			lg.Error().Msg("global config not set")
			sdk.ShowAlert(contextApp)
			return
		}
	})

	// Open and connect the SDK
	err := sdk.Open()

	if err != nil {
		lg.Panic().Err(err).Send()
	}

	// Wait until the socket is closed, or SIGTERM/SIGINT is received
	sdk.Wait()
}
