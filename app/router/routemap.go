package router

import (
	"reflect"
	"regexp"
)

var newRouteMap = map[*regexp.Regexp]any{}

var (
	argReplaceRegex = regexp.MustCompile(`(:[^\/]+)`)
)

func findHandler(path string, appCtx any) func() *Response {
	for regex, handler := range newRouteMap {
		if regex.MatchString(path) {
			argMap := make([]reflect.Value, 0)
			if appCtx != nil {
				argMap = append(argMap, reflect.ValueOf(appCtx))
			}
			for _, match := range regex.FindStringSubmatch(path)[1:] {
				argMap = append(argMap, reflect.ValueOf(match))
			}
			return func() *Response {
				val := reflect.ValueOf(handler)
				return val.Call(argMap)[0].Interface().(*Response)
			}
		}
	}
	return nil
}

func Register(path string, handler any) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func {
		logger.Error("failed to register route, handler was not a func", "path", path)
		return
	}

	// Count expected path params
	pathParamCount := len(argReplaceRegex.FindAllString(path, -1))

	// Handler may have an appCtx first param (non-string) followed by string params
	numIn := handlerType.NumIn()
	stringParamStart := 0
	if numIn > 0 && handlerType.In(0).Kind() != reflect.String {
		// First param is the context type, skip it for path param counting
		stringParamStart = 1
	}

	stringParamCount := numIn - stringParamStart
	if pathParamCount != stringParamCount {
		logger.Error("failed to register route, handler arg count did not match path", "path", path)
		return
	}

	if handlerType.NumOut() != 1 || handlerType.Out(0) != reflect.TypeFor[*Response]() {
		logger.Error("failed to register route, handler return type was not *Response", "path", path)
		return
	}

	for i := stringParamStart; i < numIn; i++ {
		if handlerType.In(i).Kind() != reflect.String {
			logger.Error("failed to register route, handler arg type was not string", "path", path)
			return
		}
	}

	pathRegex := argReplaceRegex.ReplaceAllString(path, "([^\\/]+)")
	newRouteMap[regexp.MustCompile("^"+pathRegex+"$")] = handler
}
