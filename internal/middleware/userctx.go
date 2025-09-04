package middleware

import "context"

type userKey struct{}

type UserCtx struct {
	UserID string
	Role   string
}

func WithUser(ctx context.Context, u UserCtx) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}
func FromCtx(ctx context.Context) UserCtx {
	if v := ctx.Value(userKey{}); v != nil {
		if u, ok := v.(UserCtx); ok {
			return u
		}
	}
	return UserCtx{}
}
