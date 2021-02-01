package server

import "context"

type landfillCtxKey string

func WithLandfill(ctx context.Context) context.Context {
	chErr := make(chan *FatError, 10000)
	return context.WithValue(ctx, landfillCtxKey(""), chErr)
}

func GetLandfill(ctx context.Context) chan *FatError {
	return ctx.Value(landfillCtxKey("")).(chan *FatError)
}