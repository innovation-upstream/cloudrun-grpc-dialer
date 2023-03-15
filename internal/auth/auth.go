package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/idtoken"
	grpcMetadata "google.golang.org/grpc/metadata"
)

func AuthenticateGRPCContext(
	ctx context.Context,
	addr string,
	isAuthRequired bool,
) (context.Context, error) {
	if isAuthRequired {
		splitAddr := strings.Split(addr, ":")
		tokenSource, err :=
			idtoken.NewTokenSource(ctx, fmt.Sprintf("https://%v", splitAddr[0]))
		if err != nil {
			return nil, fmt.Errorf("idtoken.NewTokenSource: %v", err)
		}
		token, err := tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("TokenSource.Token: %v", err)
		}

		ctx = grpcMetadata.NewOutgoingContext(ctx, grpcMetadata.MD{})
		ctx = grpcMetadata.AppendToOutgoingContext(
			ctx,
			"authorization",
			"Bearer "+token.AccessToken,
		)
		return ctx, nil
	}

	return ctx, nil
}
