package connection

import (
	"context"
	"fmt"

	"github.com/innovation-upstream/cloudrun-grpc-dialer/internal/auth"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type (
	CloudrunServiceName string

	AuthenticateGRPCContextFn func(context.Context) (context.Context, error)

	ServiceEndpoint struct {
		ServiceName CloudrunServiceName
		RpcEndpoint string
	}

	ServiceGRPCConnection struct {
		RpcConn     *grpc.ClientConn
		RpcEndpoint string
	}

	ServiceConnection struct {
		ServiceName CloudrunServiceName
		Connection  *ServiceGRPCConnection
	}

	ServiceConnectionList []*ServiceConnection
)

func (c *ServiceConnection) GetAuthenticateGRPCContextFn(
	isAuthRequired bool,
) AuthenticateGRPCContextFn {
	return func(ctx context.Context) (context.Context, error) {
		return c.AuthenticateGRPCContext(ctx, isAuthRequired)
	}
}

func (c *ServiceConnection) AuthenticateGRPCContext(
	ctx context.Context,
	isAuthRequired bool,
) (context.Context, error) {
	ctx, err :=
		auth.AuthenticateGRPCContext(ctx, c.Connection.RpcEndpoint, isAuthRequired)
	if err != nil {
		return ctx, errors.WithStack(err)
	}

	return ctx, nil
}

func (l ServiceConnectionList) GetConnectionForService(
	s CloudrunServiceName,
) (*ServiceConnection, error) {
	return findDialedConnectionForService(s, l)
}

func findDialedConnectionForService(
	s CloudrunServiceName,
	connList ServiceConnectionList,
) (*ServiceConnection, error) {
	if len(connList) == 0 {
		return nil, errors.New(fmt.Sprintf("no connection dialed for %v", s))
	}

	head := connList[0]
	tail := connList[1:]

	if head.ServiceName == s {
		return head, nil
	}

	return findDialedConnectionForService(s, tail)
}