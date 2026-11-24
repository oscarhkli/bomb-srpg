package server

import (
	"bomb-srpg/server/api"
	"encoding/json/v2"
	"net/http"

	"github.com/oapi-codegen/runtime"
)

type strictJSONCodec struct {
	runtime.StdJSONCodec
}

func (strictJSONCodec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

func (strictJSONCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

func InstallJSONCodec() { runtime.SetJSONCodec(strictJSONCodec{}) }

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	strict := api.NewStrictHandlerWithOptions(h, []api.StrictMiddlewareFunc{h.withToken}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  h.requestError,
		ResponseErrorHandlerFunc: h.responseError,
	})
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: h.bindError})
}
