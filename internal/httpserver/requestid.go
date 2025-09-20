package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

const reqIDkey = "req_id"

func withReqID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDkey, id)
}

func reqID(ctx context.Context) string {
	if v, ok := ctx.Value(reqIDkey).(string); ok {
		return v
	}
	return ""
}

func shortID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
