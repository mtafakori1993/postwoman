// Package scripting implements the Postman-compatible JavaScript sandbox used
// for pre-request and test scripts. Scripts run inside goja (a pure Go
// interpreter) with a curated `pm` API. No filesystem or process access is
// exposed.
package scripting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"

	"Postwoman/internal/model"
)

type Engine struct {
	mu      sync.Mutex
	timeout time.Duration
}

type Context struct {
	Request             model.Request
	Response            *model.Response
	Environment         map[string]string
	CollectionVariables map[string]string
	Globals             map[string]string
	Variables           map[string]string
	EventName           string
	Iteration           int
	RequestCount        int
}

type Result struct {
	Tests               []model.TestResult
	Console             []model.ConsoleEntry
	Environment         map[string]string
	CollectionVariables map[string]string
	Globals             map[string]string
	Variables           map[string]string
}

func NewEngine(timeout time.Duration) *Engine {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Engine{timeout: timeout}
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Run executes a script and returns assertions, console output and the final
// variable scopes. A script timeout is enforced by interrupting the VM.
func (e *Engine) Run(script string, input *Context) (*Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := &Result{
		Tests:               []model.TestResult{},
		Console:             []model.ConsoleEntry{},
		Environment:         copyMap(input.Environment),
		CollectionVariables: copyMap(input.CollectionVariables),
		Globals:             copyMap(input.Globals),
		Variables:           copyMap(input.Variables),
	}

	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	console := vm.NewObject()
	logFn := func(level string) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			args := make([]string, 0, len(call.Arguments))
			for _, arg := range call.Arguments {
				args = append(args, stringify(arg))
			}
			result.Console = append(result.Console, model.ConsoleEntry{Level: level, Args: args})
			return goja.Undefined()
		}
	}
	for _, level := range []string{"log", "info", "warn", "error", "debug"} {
		_ = console.Set(level, logFn(level))
	}
	_ = vm.Set("console", console)

	if _, err := vm.RunString(jsPrelude); err != nil {
		return result, fmt.Errorf("failed to initialise script sandbox: %w", err)
	}

	pmObj := vm.NewObject()

	_ = pmObj.Set("test", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		fn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			result.Tests = append(result.Tests, model.TestResult{Name: name, Passed: false, Error: "test callback is not a function"})
			return goja.Undefined()
		}
		_, err := fn(goja.Undefined())
		if err != nil {
			result.Tests = append(result.Tests, model.TestResult{Name: name, Passed: false, Error: cleanError(err)})
		} else {
			result.Tests = append(result.Tests, model.TestResult{Name: name, Passed: true})
		}
		return goja.Undefined()
	})

	_ = pmObj.Set("expect", func(call goja.FunctionCall) goja.Value {
		fn, _ := goja.AssertFunction(vm.Get("__makeExpectation"))
		v, err := fn(goja.Undefined(), call.Argument(0))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return v
	})

	_ = pmObj.Set("sendRequest", func(call goja.FunctionCall) goja.Value {
		req, err := buildSendRequest(call.Argument(0))
		respObj := vm.NewObject()
		if err != nil {
			_ = respObj.Set("error", err.Error())
		} else {
			httpClient := &http.Client{Timeout: 30 * time.Second}
			resp, doErr := httpClient.Do(req)
			if doErr != nil {
				_ = respObj.Set("error", doErr.Error())
			} else {
				defer resp.Body.Close()
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
				_ = respObj.Set("code", resp.StatusCode)
				_ = respObj.Set("status", http.StatusText(resp.StatusCode))
				_ = respObj.Set("headers", headerObject(resp.Header))
				_ = respObj.Set("text", string(body))
				_ = respObj.Set("json", func(goja.FunctionCall) goja.Value {
					var parsed interface{}
					if json.Unmarshal(body, &parsed) != nil {
						panic(vm.NewGoError(fmt.Errorf("response is not valid JSON")))
					}
					return vm.ToValue(parsed)
				})
			}
		}
		if cb, ok := goja.AssertFunction(call.Argument(1)); ok {
			_, _ = cb(goja.Undefined(), goja.Null(), respObj)
		}
		return respObj
	})

	scopes := map[string]*scopeState{
		"variables":           {data: result.Variables},
		"environment":         {data: result.Environment},
		"collectionVariables": {data: result.CollectionVariables},
		"globals":             {data: result.Globals},
	}
	for name, state := range scopes {
		obj := vm.NewObject()
		_ = obj.Set("get", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(state.data[call.Argument(0).String()])
		})
		_ = obj.Set("set", func(call goja.FunctionCall) goja.Value {
			state.data[call.Argument(0).String()] = call.Argument(1).String()
			return goja.Undefined()
		})
		_ = obj.Set("unset", func(call goja.FunctionCall) goja.Value {
			delete(state.data, call.Argument(0).String())
			return goja.Undefined()
		})
		_ = obj.Set("has", func(call goja.FunctionCall) goja.Value {
			_, ok := state.data[call.Argument(0).String()]
			return vm.ToValue(ok)
		})
		_ = obj.Set("clear", func(call goja.FunctionCall) goja.Value {
			for k := range state.data {
				delete(state.data, k)
			}
			return goja.Undefined()
		})
		_ = obj.Set("toObject", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(copyMap(state.data))
		})
		_ = obj.Set("replaceIn", func(call goja.FunctionCall) goja.Value {
			s := call.Argument(0).String()
			for k, v := range state.data {
				s = strings.ReplaceAll(s, "{{"+k+"}}", v)
			}
			return vm.ToValue(s)
		})
		_ = pmObj.Set(name, obj)
	}

	reqObj := vm.NewObject()
	_ = reqObj.Set("url", input.Request.URL)
	_ = reqObj.Set("method", input.Request.Method)
	_ = reqObj.Set("headers", headerMap(input.Request.Headers))
	_ = reqObj.Set("body", input.Request.Body.Raw)
	_ = pmObj.Set("request", reqObj)

	if input.Response != nil {
		respObj := vm.NewObject()
		_ = respObj.Set("code", input.Response.StatusCode)
		_ = respObj.Set("status", input.Response.StatusText)
		_ = respObj.Set("responseTime", input.Response.ResponseTime)
		_ = respObj.Set("responseSize", input.Response.BodySize)
		headers := map[string][]string{}
		for _, h := range input.Response.Headers {
			headers[h.Key] = append(headers[h.Key], h.Value)
		}
		_ = respObj.Set("headers", headers)
		_ = respObj.Set("text", func(goja.FunctionCall) goja.Value { return vm.ToValue(input.Response.Body) })
		_ = respObj.Set("json", func(goja.FunctionCall) goja.Value {
			var parsed interface{}
			if err := json.Unmarshal([]byte(input.Response.Body), &parsed); err != nil {
				panic(vm.NewGoError(fmt.Errorf("response body is not valid JSON: %w", err)))
			}
			return vm.ToValue(parsed)
		})
		fn, _ := goja.AssertFunction(vm.Get("__makeResponseExpectation"))
		expectation, _ := fn(goja.Undefined(), respObj)
		if expectation != nil {
			_ = respObj.Set("to", expectation.ToObject(vm).Get("to"))
		}
		_ = pmObj.Set("response", respObj)
	}

	cookieObj := vm.NewObject()
	_ = cookieObj.Set("get", func(call goja.FunctionCall) goja.Value {
		if input.Response == nil {
			return goja.Undefined()
		}
		name := call.Argument(0).String()
		for _, c := range input.Response.Cookies {
			if c.Name == name {
				return vm.ToValue(c.Value)
			}
		}
		return goja.Undefined()
	})
	_ = cookieObj.Set("has", func(call goja.FunctionCall) goja.Value {
		if input.Response == nil {
			return vm.ToValue(false)
		}
		name := call.Argument(0).String()
		for _, c := range input.Response.Cookies {
			if c.Name == name {
				return vm.ToValue(true)
			}
		}
		return vm.ToValue(false)
	})
	_ = pmObj.Set("cookies", cookieObj)

	infoObj := vm.NewObject()
	_ = infoObj.Set("eventName", input.EventName)
	_ = infoObj.Set("iteration", input.Iteration)
	_ = infoObj.Set("requestCount", input.RequestCount)
	_ = infoObj.Set("requestName", input.Request.Name)
	_ = pmObj.Set("info", infoObj)

	_ = vm.Set("pm", pmObj)

	timer := time.AfterFunc(e.timeout, func() {
		vm.Interrupt("script execution timed out")
	})
	defer timer.Stop()

	_, runErr := vm.RunString(script)
	vm.ClearInterrupt()

	result.Environment = scopes["environment"].data
	result.CollectionVariables = scopes["collectionVariables"].data
	result.Globals = scopes["globals"].data
	result.Variables = scopes["variables"].data

	if runErr != nil {
		return result, fmt.Errorf("%s", cleanError(runErr))
	}
	return result, nil
}

type scopeState struct {
	data map[string]string
}

func stringify(v goja.Value) string {
	if v == nil {
		return "undefined"
	}
	if obj, ok := v.(*goja.Object); ok {
		if b, err := json.Marshal(obj.Export()); err == nil {
			return string(b)
		}
	}
	return v.String()
}

func cleanError(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "Error: ")
	if idx := strings.Index(msg, " at <eval>"); idx > 0 {
		msg = msg[:idx]
	}
	return strings.TrimSpace(msg)
}

func headerMap(headers []model.KeyValue) map[string]string {
	out := map[string]string{}
	for _, h := range headers {
		if h.Enabled {
			out[h.Key] = h.Value
		}
	}
	return out
}

func headerObject(h http.Header) map[string][]string {
	out := map[string][]string{}
	for k, v := range h {
		out[k] = v
	}
	return out
}

func buildSendRequest(v goja.Value) (*http.Request, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, fmt.Errorf("pm.sendRequest requires a URL or options object")
	}
	if s, ok := v.Export().(string); ok {
		return http.NewRequest(http.MethodGet, s, nil)
	}
	obj, ok := v.(*goja.Object)
	if !ok {
		return nil, fmt.Errorf("pm.sendRequest requires a URL or options object")
	}
	urlStr := obj.Get("url").String()
	method := "GET"
	if m := obj.Get("method"); m != nil && !goja.IsUndefined(m) {
		method = strings.ToUpper(m.String())
	}
	var body io.Reader
	if b := obj.Get("body"); b != nil && !goja.IsUndefined(b) {
		if bodyObj, ok := b.(*goja.Object); ok {
			if raw := bodyObj.Get("raw"); raw != nil && !goja.IsUndefined(raw) {
				body = bytes.NewReader([]byte(raw.String()))
			}
		} else {
			body = bytes.NewReader([]byte(b.String()))
		}
	}
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	if h := obj.Get("header"); h != nil && !goja.IsUndefined(h) {
		if exported, ok := h.Export().(map[string]interface{}); ok {
			for k, val := range exported {
				req.Header.Set(k, fmt.Sprintf("%v", val))
			}
		}
	}
	return req, nil
}

const jsPrelude = `
(function () {
  function fmt(v) { try { return JSON.stringify(v); } catch (e) { return String(v); } }
  function deepEqual(a, b) {
    if (a === b) return true;
    if (typeof a !== typeof b) return false;
    if (a && b && typeof a === 'object') {
      var ka = Object.keys(a), kb = Object.keys(b);
      if (ka.length !== kb.length) return false;
      for (var i = 0; i < ka.length; i++) {
        if (!deepEqual(a[ka[i]], b[ka[i]])) return false;
      }
      return true;
    }
    return false;
  }
  function fail(msg) { var e = new Error(msg); e.__assertion = true; throw e; }

  function build(root, actual, negate) {
    function chk(cond, msg) { if (negate ? cond : !cond) fail(msg); return api; }
    root.equal = function (v) { return chk(deepEqual(actual, v), 'expected ' + fmt(actual) + ' to equal ' + fmt(v)); };
    root.eql = root.equal;
    root.equals = root.equal;
    root.above = function (v) { return chk(actual > v, 'expected ' + fmt(actual) + ' to be above ' + fmt(v)); };
    root.greaterThan = root.above;
    root.below = function (v) { return chk(actual < v, 'expected ' + fmt(actual) + ' to be below ' + fmt(v)); };
    root.lessThan = root.below;
    root.least = function (v) { return chk(actual >= v, 'expected ' + fmt(actual) + ' to be at least ' + fmt(v)); };
    root.most = function (v) { return chk(actual <= v, 'expected ' + fmt(actual) + ' to be at most ' + fmt(v)); };
    root.include = function (v) {
      var ok = Array.isArray(actual) ? actual.indexOf(v) >= 0 : String(actual).indexOf(String(v)) >= 0;
      return chk(ok, 'expected ' + fmt(actual) + ' to include ' + fmt(v));
    };
    root.contain = root.include;
    root.match = function (re) { return chk(new RegExp(re).test(String(actual)), 'expected ' + fmt(actual) + ' to match ' + re); };
    root.lengthOf = function (n) { return chk(actual != null && actual.length === n, 'expected length ' + n + ' but got ' + (actual && actual.length)); };
    root.property = function (p, v) {
      var has = actual != null && p in Object(actual);
      var ok = has && (v === undefined || deepEqual(actual[p], v));
      return chk(ok, 'expected property ' + p);
    };
    root.a = function (t) {
      var ok = (t === 'array' && Array.isArray(actual)) || typeof actual === t;
      return chk(ok, 'expected ' + fmt(actual) + ' to be a ' + t);
    };
    root.an = root.a;
    root.exist = function () { return chk(actual !== null && actual !== undefined, 'expected value to exist'); };
    root.empty = function () {
      var ok = actual == null || actual.length === 0 || Object.keys(Object(actual)).length === 0;
      return chk(ok, 'expected value to be empty');
    };
    Object.defineProperty(root, 'true', { get: function () { chk(actual === true, 'expected ' + fmt(actual) + ' to be true'); return api; } });
    Object.defineProperty(root, 'false', { get: function () { chk(actual === false, 'expected ' + fmt(actual) + ' to be false'); return api; } });
    Object.defineProperty(root, 'null', { get: function () { chk(actual === null, 'expected ' + fmt(actual) + ' to be null'); return api; } });
    Object.defineProperty(root, 'undefined', { get: function () { chk(actual === undefined, 'expected ' + fmt(actual) + ' to be undefined'); return api; } });
    Object.defineProperty(root, 'ok', { get: function () { chk(!!actual, 'expected ' + fmt(actual) + ' to be truthy'); return api; } });
    Object.defineProperty(root, 'be', { get: function () { return root; } });
    Object.defineProperty(root, 'have', { get: function () { return root; } });
    return root;
  }

  var api = {};
  globalThis.__makeExpectation = function (actual) {
    api = {};
    api.to = build({}, actual, false);
    api.not = build({}, actual, true);
    api.to.not = api.not;
    return api;
  };

  globalThis.__makeResponseExpectation = function (resp) {
    var api = { to: {} };
    var have = {};
    have.status = function (code) { if (resp.code !== code) fail('expected status ' + code + ' but got ' + resp.code); return api; };
    have.header = function (name) {
      var found = false;
      if (resp.headers) { for (var k in resp.headers) { if (k.toLowerCase() === String(name).toLowerCase()) found = true; } }
      if (!found) fail('expected header ' + name);
      return api;
    };
    have.body = function (expected) { if (String(resp.text()) !== String(expected)) fail('unexpected body'); return api; };
    have.jsonBody = function (path, value) {
      var body = resp.json();
      if (path) {
        var parts = String(path).replace(/\[(\d+)\]/g, '.$1').split('.');
        for (var i = 0; i < parts.length; i++) { if (parts[i] === '') continue; body = body[parts[i]]; }
      }
      if (!deepEqual(body, value)) fail('expected JSON body at ' + path + ' to equal ' + fmt(value));
      return api;
    };
    api.to.have = have;
    Object.defineProperty(api.to, 'be', { get: function () {
      return {
        get ok() { if (!(resp.code >= 200 && resp.code < 300)) fail('expected successful status'); return api; },
        get success() { if (!(resp.code >= 200 && resp.code < 300)) fail('expected success'); return api; },
        get error() { if (!(resp.code >= 400)) fail('expected error status'); return api; },
        get clientError() { if (!(resp.code >= 400 && resp.code < 500)) fail('expected client error'); return api; },
        get serverError() { if (!(resp.code >= 500)) fail('expected server error'); return api; },
        get redirected() { if (!(resp.code >= 300 && resp.code < 400)) fail('expected redirect'); return api; }
      };
    }});
    return api;
  };
})();
`
