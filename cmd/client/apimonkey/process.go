package main

import (
	"context"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
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

			lg.Debug().Msgf("got raw response %v", resp.String())
			parsed, err := fastjson.ParseBytes(resp.Bytes())
			if err != nil {
				sdk.ShowAlert(contextApp)
				lg.Err(errors.Wrap(err, "error parsing request")).Send()
				return
			}

			value := parsed.String()
			if globalConfig.ResponseJSONSelector != "" {
				selectorVal := parsed.Get(globalConfig.ResponseJSONSelector)
				if selectorVal.Type() == fastjson.TypeString {
					if s, errSBytes := selectorVal.StringBytes(); errSBytes != nil {
						sdk.ShowAlert(contextApp)
						lg.Err(errors.Wrap(err, "error parsing StringBytes")).Send()
						return
					} else {
						value = string(s)
					}
				} else {
					value = selectorVal.String()
				}
			}

			lg.Debug().Msgf("got raw value %v", string(value))

			handleResponse(value)
		}()

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func handleResponse(response string) {
	lg.Debug().Msgf("mapper: %v", spew.Sdump(globalConfig.ResponseMapper))
	if globalConfig.ResponseMapper == nil || globalConfig.ResponseMapper[response] == "" {
		sdk.SetTitle(contextApp, response, 0)
		sdk.ShowOk(contextApp)

		return
	}

	mapped, ok := globalConfig.ResponseMapper[response]
	if !ok { // should not happen
		lg.Error().Msgf("response mapper not found for value - %v", response)
		sdk.ShowAlert(contextApp)
		return
	}

	if strings.HasPrefix("http", mapped) || strings.HasSuffix(".png", mapped) {

	} else {
		sdk.SetTitle(contextApp, mapped, 0)
		sdk.ShowOk(contextApp)

		return
	}
}
