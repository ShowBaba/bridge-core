package endpoint

import (
	"context"
	"errors"
	"time"

	"github.com/dop251/goja"
	"github.com/google/uuid"
)

var (
	errScriptTimeout = errors.New("script timed out")
)

type preInput struct {
	Req any `json:"req"`
	Env any `json:"env"`
}
type postInput struct {
	Res any `json:"res"`
	Req any `json:"req"`
	Env any `json:"env"`
}

type scriptResult struct {
	Abort  *scriptAbort  `json:"abort"`
	Mutate *scriptMutate `json:"mutate"`
}
type scriptAbort struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}
type scriptMutate struct {
	Headers    map[string]string   `json:"headers"`
	PathParams map[string]string   `json:"pathParams"`
	Query      map[string][]string `json:"query"`
	Body       any                 `json:"body"`
	Values     []any               `json:"values"`

	Status *int `json:"status"`
}

// runJSScript executes user JS with injected globals:
// - Pre script:  globals { req, env }
// - Post script: globals { res, req, env }
// Also exposes helper functions:
//
//	abort(message: string, status?: number)
//	mutate(patch: { headers?, pathParams?, query?, body?, values?, status? })
func runJSScript(ctx context.Context, code string, input any, timeoutMS int) (*scriptResult, error) {
	if code == "" {
		return &scriptResult{}, nil
	}
	if timeoutMS <= 0 {
		timeoutMS = 200
	}

	vm := goja.New()
	out := &scriptResult{}

	mergeMutate := func(patch map[string]any) {
		if out.Mutate == nil {
			out.Mutate = &scriptMutate{}
		}
		if v, ok := patch["headers"]; ok && v != nil {
			var m map[string]string
			_ = vm.ExportTo(vm.ToValue(v), &m)
			if len(m) > 0 {
				if out.Mutate.Headers == nil {
					out.Mutate.Headers = map[string]string{}
				}
				for k, vv := range m {
					out.Mutate.Headers[k] = vv
				}
			}
		}
		if v, ok := patch["pathParams"]; ok && v != nil {
			var m map[string]string
			_ = vm.ExportTo(vm.ToValue(v), &m)
			if len(m) > 0 {
				if out.Mutate.PathParams == nil {
					out.Mutate.PathParams = map[string]string{}
				}
				for k, vv := range m {
					out.Mutate.PathParams[k] = vv
				}
			}
		}
		if v, ok := patch["query"]; ok && v != nil {
			var m map[string][]string
			_ = vm.ExportTo(vm.ToValue(v), &m)
			if len(m) > 0 {
				if out.Mutate.Query == nil {
					out.Mutate.Query = map[string][]string{}
				}
				for k, vv := range m {
					out.Mutate.Query[k] = vv
				}
			}
		}
		if v, ok := patch["body"]; ok {
			out.Mutate.Body = v
		}
		if v, ok := patch["values"]; ok && v != nil {
			var arr []any
			_ = vm.ExportTo(vm.ToValue(v), &arr)
			if len(arr) > 0 {
				out.Mutate.Values = arr
			}
		}
		if v, ok := patch["status"]; ok && v != nil {
			var st int
			_ = vm.ExportTo(vm.ToValue(v), &st)
			out.Mutate.Status = &st
		}
	}

	// abort(message: string, status?: number)
	_ = vm.Set("abort", func(call goja.FunctionCall) goja.Value {
		msg := ""
		if len(call.Arguments) > 0 {
			msg = call.Arguments[0].String()
		}
		status := 400
		if len(call.Arguments) > 1 {
			if n := int(call.Arguments[1].ToInteger()); n > 0 {
				status = n
			}
		}
		out.Abort = &scriptAbort{Status: status, Message: msg}
		return goja.Undefined()
	})

	// mutate(patch: { headers?, pathParams?, query?, body?, values?, status? })
	_ = vm.Set("mutate", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			return goja.Undefined()
		}
		var patch map[string]any
		_ = vm.ExportTo(call.Arguments[0], &patch)
		if patch != nil {
			mergeMutate(patch)
		}
		return goja.Undefined()
	})

	// Helper: set a property, wrapping Go funcs so they’re callable in JS
	setProp := func(obj *goja.Object, key string, val any) {
		switch fn := val.(type) {
		case func() string:
			_ = obj.Set(key, func() string { return fn() })
		case func() any:
			_ = obj.Set(key, func() any { return fn() })
		default:
			_ = obj.Set(key, val)
		}
	}

	// Always build a real JS object for env and attach now()/uuid()
	injectEnv := func(env any) {
		obj := vm.NewObject()

		// Defaults to guarantee presence
		_ = obj.Set("now", func() string { return time.Now().UTC().Format(time.RFC3339Nano) })
		_ = obj.Set("uuid", func() string { return uuid.NewString() })

		// Copy provided fields over
		switch m := env.(type) {
		case map[string]any:
			for k, v := range m {
				setProp(obj, k, v)
			}

		}

		_ = vm.Set("env", obj)
	}

	// Inject globals: req/res/env
	switch v := input.(type) {
	case preInput:
		_ = vm.Set("req", v.Req)
		injectEnv(v.Env)
	case postInput:
		_ = vm.Set("res", v.Res)
		_ = vm.Set("req", v.Req)
		injectEnv(v.Env)
	default:
		// no-op
	}

	done := make(chan error, 1)
	go func() {
		_, err := vm.RunString(code)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Duration(timeoutMS) * time.Millisecond):
		return nil, errScriptTimeout
	case err := <-done:
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func jsEnv(version, appSlug string) map[string]any {
	return map[string]any{
		"version": version,
		"appSlug": appSlug,
		"now": func() string {
			return time.Now().UTC().Format(time.RFC3339Nano)
		},
		"uuid": func() string {
			return uuid.NewString()
		},
	}
}
