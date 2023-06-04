package mwwrapper

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type (
	MiddlewareGroupKey int
	MiddlewareGroups   map[MiddlewareGroupKey]*chi.Middlewares
)

type MiddlewareWrapper struct {
	Groups *MiddlewareGroups
}

type Options struct {
	Groups *MiddlewareGroups
}

func NewMiddleWareWrapper(o Options) MiddlewareWrapper {
	return MiddlewareWrapper{Groups: o.Groups}
}

func (mwg MiddlewareGroups) GetChain(name MiddlewareGroupKey) []func(http.Handler) http.Handler {
	mw, ok := mwg[name]
	if !ok {
		panic("not found middleware group")
	}

	return *mw
}
