package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
	"meow.tf/streamdeck/sdk"
)

type Instance struct {
	cfg        *config
	contextApp string
	lg         zerolog.Logger
}

func (i *Instance) SetConfig(ctxId string, cfg *config) {
	if i.contextApp != ctxId {
		return
	}

	i.cfg = cfg
	lg.Debug().Msg("set config")
}

func (i *Instance) ShowAlert() {
	sdk.ShowAlert(i.contextApp)
}

func (i *Instance) KeyPressed() {
	if i.cfg == nil {
		lg.Error().Msg("global config not set")
		sdk.ShowAlert(i.contextApp)
		return
	}

	targetUrl := i.cfg.BrowserUrl
	if targetUrl == "" {
		targetUrl = i.cfg.ApiUrl
	}

	targetUrl = runTemplate(targetUrl, i.cfg)

	if err := exec.Command("rundll32",
		"url.dll,FileProtocolHandler", targetUrl).Start(); err != nil {

		lg.Error().Msg("global config not set")
		sdk.ShowAlert(i.contextApp)
		return
	}
}

func (i *Instance) Run() {
	for context.Background().Err() == nil {
		interval := 30
		if i.cfg.IntervalSeconds > 0 {
			interval = i.cfg.IntervalSeconds
		}

		func() {
			apiUrl := runTemplate(i.cfg.ApiUrl, i.cfg)
			lg.Debug().Msgf("sending request to %v", apiUrl)
			httpReq := req.C().NewRequest()

			for k, v := range i.cfg.Headers {
				httpReq.SetHeader(k, runTemplate(v, i.cfg))
			}

			resp, err := httpReq.Get(apiUrl)
			if err != nil {
				sdk.ShowAlert(i.contextApp)
				lg.Err(errors.Wrap(err, "error sending request")).Send()
				return
			}

			lg.Debug().Msgf("got raw response %v", resp.String())
			if err != nil {
				sdk.ShowAlert(i.contextApp)
				lg.Err(errors.Wrap(err, "error parsing request")).Send()
				return
			}

			value := resp.String()
			if i.cfg.ResponseJSONSelector != "" {
				selectorVal := gjson.Get(resp.String(), i.cfg.ResponseJSONSelector)

				if selectorVal.Type == gjson.Null {
					sdk.ShowAlert(i.contextApp)
					lg.Err(errors.New("no data found by ResponseJSONSelector")).Send()
					return
				}

				value = selectorVal.String()

				if value == "" {
					sdk.ShowAlert(i.contextApp)
					lg.Err(errors.Wrap(err, "empty value got from ResponseJSONSelector")).Send()
				}
			}

			lg.Debug().Msgf("got raw value %v", value)

			i.handleResponse(value)
		}()

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (i *Instance) handleResponse(response string) {
	var sb strings.Builder
	prefix := runTemplate(i.cfg.TitlePrefix, i.cfg)
	if prefix != "" {
		sb.WriteString(strings.ReplaceAll(prefix, "\\n", "\n") + "\n")
	}

	if len(i.cfg.ResponseMapper) == 0 {
		if response == "" {
			sb.WriteString("!! NO !!\n !! MAPPING !!")
		} else {
			sb.WriteString(response)
		}

		sdk.SetTitle(i.contextApp, sb.String(), 0)
		sdk.SetImage(i.contextApp, "", 0)
		sdk.ShowOk(i.contextApp)

		return
	}

	mapped, ok := i.cfg.ResponseMapper[response]
	def, defaultOk := i.cfg.ResponseMapper["*"]

	if !ok && defaultOk {
		mapped = def
	} else if !ok && !defaultOk {
		sb.WriteString("!! NO !!\n !! MAPPING !!")
		lg.Error().Msgf("response mapper not found for value - %v", response)

		sdk.SetTitle(i.contextApp, sb.String(), 0)
		sdk.SetImage(i.contextApp, "", 0)
		sdk.ShowAlert(i.contextApp)

		return
	}

	if strings.HasPrefix(mapped, "http") || strings.HasSuffix(mapped, ".png") || strings.HasSuffix(mapped, ".svg") {
		if sb.Len() > 0 {
			sdk.SetTitle(i.contextApp, sb.String(), 0)
		}

		if strings.HasSuffix(mapped, ".png") || strings.HasSuffix(mapped, ".svg") {
			fileData, err := readFile(filepath.Join("images", mapped))

			if err != nil {
				lg.Err(errors.Wrap(err, "image file not found")).Send()
				sdk.SetImage(i.contextApp, "", 0)
				sdk.ShowAlert(i.contextApp)
				return
			}

			imageData := ""
			if strings.HasSuffix(mapped, ".png") {
				imageData = fmt.Sprintf("data:image/png;base64, %v", base64.StdEncoding.EncodeToString(fileData))
			} else if strings.HasSuffix(mapped, ".svg") {
				imageData = fmt.Sprintf("data:image/svg+xml;charset=utf8,%v", string(fileData))
			}

			sdk.SetImage(i.contextApp, imageData, 0)
		}

		sdk.ShowOk(i.contextApp)
	} else {
		sb.WriteString(mapped)
		sdk.SetTitle(i.contextApp, sb.String(), 0)
		sdk.SetImage(i.contextApp, "", 0)
		sdk.ShowOk(i.contextApp)

		return
	}
}
