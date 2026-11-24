package server

import (
	"bomb-srpg/server/api"
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
)

var errBodyAlreadyBound = errors.New("body already bound")

// validatedBody is the type the generated decoder's runtime.BindJSONBody binds
// from, with exact member names. Read fails so it never falls back to a lenient decode.
type validatedBody struct {
	io.Closer
	value any
}

func (validatedBody) Read([]byte) (int, error) { return 0, errBodyAlreadyBound }
func (b validatedBody) ValidatedValue() any    { return b.value }

func (h *Handler) strictJSONBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Pattern {
		case "POST /api/match-rooms/{roomID}/match",
			"POST /api/match-rooms/{roomID}/match/turn-commands",
			"POST /api/match-rooms/{roomID}/match/surrender":
			data, err := io.ReadAll(r.Body)
			var value any
			if err == nil {
				err = json.UnmarshalRead(bytes.NewReader(data), &value)
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
