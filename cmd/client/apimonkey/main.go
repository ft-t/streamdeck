package main

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/valyala/fastjson"
	"gopkg.in/natefinch/lumberjack.v2"
	"meow.tf/streamdeck/sdk"

	"github.com/ft-t/streamdeck/cmd/scripts"
)

var instances = map[string]*Instance{}
var mut sync.Mutex

func setSettingsFromPayload(
	payload *fastjson.Value,
	ctxId string,
	instance *Instance,
) {
	logger := instance.GetLogger()

	if instance == nil {
		logger.Warn().Msgf("instance %v not found", ctxId)
		return
	}

	settingsBytes := payload.MarshalTo(nil)
	logger.Trace().Msgf("Got configuration: %v", string(settingsBytes))
	var tempConfig config

	if err := json.Unmarshal(settingsBytes, &tempConfig); err != nil {
		logger.Err(err).Send()
		instance.ShowAlert()
		return
	}

	instance.SetConfig(ctxId, &tempConfig)
}

func main() {
	logFile := &lumberjack.Logger{
		Filename:   "logs/log.log",
		MaxSize:    30,
		MaxBackups: 3,
		MaxAge:     10,
		Compress:   false,
	}

	lg := zerolog.New(zerolog.MultiLevelWriter(os.Stdout, logFile)).With().Timestamp().Logger()

	sdk.AddHandler(func(event *sdk.WillAppearEvent) {
		if event.Payload == nil {
			return
		}

		mut.Lock()
		defer mut.Unlock()
		instance, ok := instances[event.Context]

		if !ok {
			instance = NewInstance(event.Context, scripts.NewLua(), lg)
			instances[event.Context] = instance
		}

		setSettingsFromPayload(event.Payload.Get("settings"), event.Context, instance)
		instance.StartAsync()
	})

	sdk.AddHandler(func(event *sdk.WillDisappearEvent) {
		if event.Payload == nil {
			return
		}

		mut.Lock()
		defer mut.Unlock()
		instance, ok := instances[event.Context]
		if !ok {
			return
		}

		instance.StartAsync()
	})

	sdk.AddHandler(func(event *sdk.ReceiveSettingsEvent) {
		lg.Debug().Msg("got ReceiveSettingsEvent")
		setSettingsFromPayload(event.Settings, event.Context, instances[event.Context])
	})

	sdk.AddHandler(func(event *sdk.KeyDownEvent) {
		instance, ok := instances[event.Context]
		if !ok {
			lg.Warn().Msgf("instance %v not found", event.Context)
		}

		instance.KeyPressed()
	})

	// Open and connect the SDK
	err := sdk.Open()

	if err != nil {
		lg.Panic().Err(err).Send()
	}
	// run

	// Wait until the socket is closed, or SIGTERM/SIGINT is received
	sdk.Wait()
}
