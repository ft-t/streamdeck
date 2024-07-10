package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/imroc/req/v3"
	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
	"meow.tf/streamdeck/sdk"
)

type Instance struct {
	cfg           *config
	contextApp    string
	currentLogger zerolog.Logger
	executor      ScriptExecutor
	ctx           context.Context
	ctxCancel     context.CancelFunc
	mut           sync.Mutex
}

func NewInstance(
	contextApp string,
	executor ScriptExecutor,
	logger zerolog.Logger,
) *Instance {
	return &Instance{
		contextApp:    contextApp,
		currentLogger: logger.With().Str("context_id", contextApp).Logger(),
		executor:      executor,
		mut:           sync.Mutex{},
	}
}

func (i *Instance) GetLogger() zerolog.Logger {
	return i.currentLogger
}

func (i *Instance) SetConfig(ctxId string, cfg *config) {
	if i.contextApp != ctxId {
		return
	}

	i.cfg = cfg

	if i.cfg.MinLogLevel != nil {
		i.currentLogger = i.currentLogger.Level(*i.cfg.MinLogLevel)
	} else {
		i.currentLogger = i.currentLogger.Level(zerolog.WarnLevel)
	}
}

func (i *Instance) ShowAlert() {
	sdk.ShowAlert(i.contextApp)
}

func (i *Instance) KeyPressed() {
	if i.cfg == nil {
		i.currentLogger.Error().Msg("global config not set")
		sdk.ShowAlert(i.contextApp)
		return
	}

	targetUrl := i.cfg.BrowserUrl
	if targetUrl == "" {
		targetUrl = i.cfg.ApiUrl
	}

	targetUrl = runTemplate(targetUrl, i.cfg, i.currentLogger)

	if err := exec.Command("rundll32",
		"url.dll,FileProtocolHandler", targetUrl).Start(); err != nil {

		i.currentLogger.Error().Msg("global config not set")
		sdk.ShowAlert(i.contextApp)
		return
	}
}

func (i *Instance) StartAsync() {
	i.mut.Lock()
	if i.ctxCancel != nil { // first cancel old routine
		i.ctxCancel()
	}

	i.ctx, i.ctxCancel = context.WithCancel(context.Background())
	i.mut.Unlock()

	go i.run()
}

func (i *Instance) Stop() {
	i.mut.Lock()

	if i.ctxCancel != nil { // first cancel old routine
		i.ctxCancel()
	}
	i.ctxCancel = nil

	i.mut.Unlock()
}

func (i *Instance) run() {
	ctx := i.ctx

	for ctx.Err() == nil {
		interval := 30
		if i.cfg.IntervalSeconds > 0 {
			interval = i.cfg.IntervalSeconds
		}

		innerCtx, innerCancel := context.WithCancel(ctx)
		innerCtx = i.currentLogger.WithContext(innerCtx)

		processErr := i.sendAndProcess(innerCtx)
		innerCancel()

		if processErr != nil {
			i.currentLogger.Err(errors.Wrap(processErr, "error processing response")).Send()
			i.ShowAlert()
		} else {
			if i.cfg.ShowSuccessNotification {
				sdk.ShowOk(i.contextApp)
			}
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (i *Instance) sendAndProcess(ctx context.Context) error {
	apiUrl := runTemplate(i.cfg.ApiUrl, i.cfg, i.currentLogger)
	httpReq := req.C().NewRequest()
	httpReq = httpReq.SetContext(ctx)

	for k, v := range i.cfg.Headers {
		httpReq.SetHeader(k, runTemplate(v, i.cfg, i.currentLogger))
	}

	zerolog.Ctx(ctx).Trace().Str("url", apiUrl).Msg("sending request")
	resp, err := httpReq.Get(apiUrl)
	if err != nil {
		return errors.Wrap(err, "error sending request")
	}

	value := resp.String()
	zerolog.Ctx(ctx).Trace().Str("response", value).Msg("got raw response")

	if strings.TrimSpace(i.cfg.BodyScript) != "" {
		zerolog.Ctx(ctx).Trace().Str("script", i.cfg.BodyScript).Msg("executing script")

		scriptResult, scriptErr := i.executor.Execute(
			ctx,
			i.cfg.BodyScript,
			value,
			resp.StatusCode,
			i.cfg.Headers,
			i.cfg.TemplateParameters,
		)
		if scriptErr != nil {
			return errors.Wrap(scriptErr, "error executing script")
		}

		value = scriptResult

		zerolog.Ctx(ctx).Trace().Str("result", value).Msg("script executed")
	}

	zerolog.Ctx(ctx).Debug().
		Str("response", value).
		Str("selector", i.cfg.ResponseJSONSelector).
		Msg("post script processing")

	if i.cfg.ResponseJSONSelector != "" {
		selectorVal := gjson.Get(value, i.cfg.ResponseJSONSelector)

		if selectorVal.Type == gjson.Null {
			return errors.New("no data found by ResponseJSONSelector")
		}

		value = selectorVal.String()

		if value == "" {
			return errors.New("empty value got from ResponseJSONSelector")
		}
	}

	zerolog.Ctx(ctx).Info().Str("final_result", value).Msgf("final")

	return i.handleResponse(ctx, value)
}

func (i *Instance) handleResponse(_ context.Context, response string) error {
	var sb strings.Builder
	prefix := runTemplate(i.cfg.TitlePrefix, i.cfg, i.currentLogger)
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

		return nil
	}

	mapped, ok := i.cfg.ResponseMapper[response]
	def, defaultOk := i.cfg.ResponseMapper["*"]

	if !ok && defaultOk {
		mapped = def
	} else if !ok && !defaultOk {
		sb.WriteString("!! NO !!\n !! MAPPING !!")
		i.currentLogger.Error().Msgf("response mapper not found for value - %v", response)

		sdk.SetTitle(i.contextApp, sb.String(), 0)
		sdk.SetImage(i.contextApp, "", 0)

		return errors.Newf("response mapper not found for value - %v", response)
	}

	if strings.HasPrefix(mapped, "http") || strings.HasSuffix(mapped, ".png") || strings.HasSuffix(mapped, ".svg") {
		if sb.Len() > 0 {
			sdk.SetTitle(i.contextApp, sb.String(), 0)
		}

		if strings.HasSuffix(mapped, ".png") || strings.HasSuffix(mapped, ".svg") {
			fileData, err := readFile(filepath.Join("images", mapped))

			if err != nil {
				sdk.SetImage(i.contextApp, "", 0)
				return errors.Join(err, errors.New("image file not found"))
			}

			imageData := ""
			if strings.HasSuffix(mapped, ".png") {
				imageData = fmt.Sprintf("data:image/png;base64, %v", base64.StdEncoding.EncodeToString(fileData))
			} else if strings.HasSuffix(mapped, ".svg") {
				imageData = fmt.Sprintf("data:image/svg+xml;charset=utf8,%v", string(fileData))
			}

			sdk.SetImage(i.contextApp, imageData, 0)
		}
	} else {
		sb.WriteString(mapped)
		sdk.SetTitle(i.contextApp, sb.String(), 0)
		sdk.SetImage(i.contextApp, "", 0)
	}

	return nil
}
