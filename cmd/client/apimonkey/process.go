package main

import (
	"context"
	"time"

	"github.com/imroc/req/v3"
	"github.com/pkg/errors"
	"github.com/valyala/fastjson"
	"meow.tf/streamdeck/sdk"
)

var isJobRunning = false

func process() {
	if isJobRunning {
		return
	}

	isJobRunning = true

	for context.Background().Err() == nil {
		interval := 30
		if globalConfig.IntervalSeconds > 0 {
			interval = globalConfig.IntervalSeconds
		}

		func() {
			lg.Debug().Msgf("sending request to %v", globalConfig.ApiUrl)
			resp, err := req.C().NewRequest().Get(globalConfig.ApiUrl)

			if err != nil {
				sdk.ShowAlert(contextApp)
				lg.Err(errors.Wrap(err, "error sending request")).Send()
				return
			}

			parsed, err := fastjson.ParseBytes(resp.Bytes())
			if err != nil {
				sdk.ShowAlert(contextApp)
				lg.Err(errors.Wrap(err, "error parsing request")).Send()
				return
			}

			value := parsed.String()
			if globalConfig.ResponseJSONSelector != "" {
				value = parsed.Get(globalConfig.ResponseJSONSelector).String()
			}

			handleResponse(value)
		}()

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func handleResponse(response string) {
	if globalConfig.ResponseMapper == nil || globalConfig.ResponseMapper[response] == "" {
		sdk.SetTitle(contextApp, response, 0)
		sdk.ShowOk(contextApp)

		return
	}
}
