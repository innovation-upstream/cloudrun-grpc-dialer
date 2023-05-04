package auth

import "context"

type AuthenticateGRPCContextFn func(context.Context) (context.Context, error)
