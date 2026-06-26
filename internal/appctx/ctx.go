package appctx

import "context"

var Ctx context.Context

func Set(ctx context.Context) {
	Ctx = ctx
}
