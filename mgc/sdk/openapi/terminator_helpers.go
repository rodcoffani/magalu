package openapi

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

var terminateTemplateStrings = []string{
	"finished",
	"terminated",
	"true",
}

func jsonPathTerminationCheck(wt *waitTermination, exec core.Executor, logger *zap.SugaredLogger) (core.TerminatorExecutor, error) {
	builder := gval.Full(jsonpath.PlaceholderExtension())
	jp, err := builder.NewEvaluable(wt.JSONPathQuery)
	if err == nil {
		tExec := core.NewTerminatorExecutorWithCheck(exec, wt.MaxRetries, wt.IntervalInSeconds, func(ctx context.Context, exec core.Executor, result core.ResultWithValue) (bool, error) {
			data := map[string]any{
				"result":     result.Value(),
				"parameters": result.Source().Parameters,
				"configs":    result.Source().Configs,
			}
			v, err := jp(ctx, data)
			if err != nil {
				logger.Warnw("error evaluating jsonpath query", "query", wt.JSONPathQuery, "target", data, "error", err)
				return false, err
			}

			logger.Debugf("jsonpath expression %#v data is %#v", wt.JSONPathQuery, data)
			if v == nil {
				return false, nil
			} else if lst, ok := v.([]any); ok {
				return len(lst) > 0, nil
			} else if m, ok := v.(map[string]any); ok {
				return len(m) > 0, nil
			} else if b, ok := v.(bool); ok {
				return b, nil
			} else {
				logger.Warnw("unknown jsonpath result. Expected list, map or boolean", "data", data)
				return false, fmt.Errorf("unknown jsonpath data. Expected list, map or boolean. Got %+v", v)
			}
		})
		return tExec, nil
	} else {
		logger.Warnw("error parsing jsonpath. Executing without polling", "expression", wt.JSONPathQuery, "error", err)
		return nil, err
	}
}

func templateTerminationCheck(wt *waitTermination, exec core.Executor, logger *zap.SugaredLogger) (core.TerminatorExecutor, error) {
	tmpl, err := template.New("core.wait-termination").Parse(wt.TemplateQuery)
	if err != nil {
		return nil, err
	}

	tExec := core.NewTerminatorExecutorWithCheck(exec, wt.MaxRetries, wt.IntervalInSeconds, func(ctx context.Context, exec core.Executor, result core.ResultWithValue) (bool, error) {
		data := map[string]any{
			"result":     result.Value(),
			"parameters": result.Source().Parameters,
			"configs":    result.Source().Configs,
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, data)
		if err != nil {
			logger.Warnw("error evaluating template query", "query", wt.TemplateQuery, "target", data, "error", err)
			return false, err
		}

		logger.Debugf("template expression %#v data is %#v", wt.TemplateQuery, data)
		s := buf.String()
		s = strings.Trim(s, " \t\n\r")
		return slices.Contains(terminateTemplateStrings, s), nil
	})
	return tExec, nil
}

func wrapInTerminatorExecutor(logger *zap.SugaredLogger, wtExt map[string]any, exec core.Executor) (core.TerminatorExecutor, error) {
	wt := &waitTermination{}
	if err := utils.DecodeValue(wtExt, wt); err != nil {
		logger.Warnw("error decoding extension wait-termination", "data", wtExt, "error", err)
	}

	if wt.MaxRetries <= 0 {
		wt.MaxRetries = defaultWaitTermination.MaxRetries
	}
	if wt.IntervalInSeconds <= 0 {
		wt.IntervalInSeconds = defaultWaitTermination.IntervalInSeconds
	}

	if wt.JSONPathQuery != "" {
		return jsonPathTerminationCheck(wt, exec, logger)
	}

	if wt.TemplateQuery != "" {
		return templateTerminationCheck(wt, exec, logger)
	}

	return nil, fmt.Errorf("no termination check expression provided")
}