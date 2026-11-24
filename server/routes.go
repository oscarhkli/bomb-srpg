package server

import (
	"bomb-srpg/server/api"
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
)

var (
	errBodyAlreadyBound = errors.New("body already bound")
	errTrailingData     = errors.New("trailing data after value")
	errNegativeUnitID   = errors.New("unitId is negative zero")
)

type notWholeNumber struct{}

const (
	createMatchPattern  = "POST /api/match-rooms/{roomID}/match"
	turnCommandsPattern = "POST /api/match-rooms/{roomID}/match/turn-commands"
	surrenderPattern    = "POST /api/match-rooms/{roomID}/match/surrender"
)

// validatedBody is the type the generated decoder's runtime.BindJSONBody binds
// from, with exact member names. Read fails so it never falls back to a lenient decode.
type validatedBody struct {
	io.Closer
	value any
}

func (validatedBody) Read([]byte) (int, error) { return 0, errBodyAlreadyBound }
func (b validatedBody) ValidatedValue() any    { return b.value }

func wholeNumber(text string) any {
	if text == "-0" {
		return math.Copysign(0, -1)
	}
	if n, err := strconv.ParseInt(text, 10, 64); err == nil {
		return n
	}
	return notWholeNumber{}
}

func readJSONObject(dec *jsontext.Decoder) (any, error) {
	obj := map[string]any{}
	for dec.PeekKind() != jsontext.KindEndObject {
		name, err := dec.ReadToken()
		if err != nil {
			return nil, err
		}
		if obj[name.String()], err = readJSONValue(dec); err != nil {
			return nil, err
		}
	}
	_, err := dec.ReadToken()
	return obj, err
}

func readJSONArray(dec *jsontext.Decoder) (any, error) {
	arr := []any{}
	for dec.PeekKind() != jsontext.KindEndArray {
		item, err := readJSONValue(dec)
		if err != nil {
			return nil, err
		}
		arr = append(arr, item)
	}
	_, err := dec.ReadToken()
	return arr, err
}

func readJSONValue(dec *jsontext.Decoder) (any, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	switch tok.Kind() {
	case jsontext.KindTrue:
		return true, nil
	case jsontext.KindFalse:
		return false, nil
	case jsontext.KindString:
		return tok.String(), nil
	case jsontext.KindNumber:
		return wholeNumber(tok.String()), nil
	case jsontext.KindBeginObject:
		return readJSONObject(dec)
	case jsontext.KindBeginArray:
		return readJSONArray(dec)
	case jsontext.KindNull, jsontext.KindInvalid, jsontext.KindEndObject, jsontext.KindEndArray:
		return nil, nil
	}
	return nil, nil
}

func decodeStrictJSON(pattern string, data []byte) (any, error) {
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	value, err := readJSONValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.ReadToken(); !errors.Is(err, io.EOF) {
		return nil, errors.Join(errTrailingData, err)
	}
	if members, ok := value.(map[string]any); ok && pattern == turnCommandsPattern {
		if id, ok := members["unitId"].(float64); ok && math.Signbit(id) {
			return nil, errNegativeUnitID
		}
	}
	return value, nil
}

func (h *Handler) strictJSONBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Pattern {
		case createMatchPattern, turnCommandsPattern, surrenderPattern:
			data, err := io.ReadAll(r.Body)
			var value any
			if err == nil {
				value, err = decodeStrictJSON(r.Pattern, data)
			}
			if err != nil {
				h.requestError(w, r, err)
				return
			}
			r.Body = validatedBody{Closer: r.Body, value: value}
		}
		next.ServeHTTP(w, r)
	})
}

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	strict := api.NewStrictHandlerWithOptions(h, []api.StrictMiddlewareFunc{h.withToken}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  h.requestError,
		ResponseErrorHandlerFunc: h.responseError,
	})
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: h.bindError,
		Middlewares:      []api.MiddlewareFunc{h.strictJSONBody},
	})
}
