package main

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/valyala/fastjson"
	"gopkg.in/natefinch/lumberjack.v2"
	"meow.tf/streamdeck/sdk"
)

var lg zerolog.Logger

var instances = map[string]*Instance{}
var mut sync.Mutex

//var contextApp string
//var globalConfig *config

func setSettingsFromPayload(payload *fastjson.Value, ctxId string, instance *Instance) {
	if instance == nil {
		lg.Warn().Msgf("instance %v not found", ctxId)
	}
	settingsBytes := payload.MarshalTo(nil)
	lg.Debug().Msgf("Got configuration: %v", string(settingsBytes))
	var tempConfig config

	if err := json.Unmarshal(settingsBytes, &tempConfig); err != nil {
		lg.Err(err).Send()
		instance.ShowAlert()
		return
	}

	instance.SetConfig(ctxId, &tempConfig)
}

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
		if event.Payload == nil {
			return
		}

		mut.Lock()
		defer mut.Unlock()
		instance, _ := instances[event.Context]

		if instance == nil {
			instance = &Instance{
				contextApp: event.Context,
				lg:         lg.With().Str("context_id", event.Context).Logger(),
			}

			instances[event.Context] = instance
		}

		setSettingsFromPayload(event.Payload.Get("settings"), event.Context, instance)
		go instance.Run()
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

	// Wait until the socket is closed, or SIGTERM/SIGINT is received
	sdk.Wait()
}
