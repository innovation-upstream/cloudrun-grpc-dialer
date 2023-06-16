package connection

import (
	"context"
	"fmt"

	"github.com/innovation-upstream/cloudrun-grpc-dialer/auth"
	internalAuth "github.com/innovation-upstream/cloudrun-grpc-dialer/internal/auth"
	"github.com/innovation-upstream/cloudrun-grpc-dialer/service"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type (
	ServiceGRPCConnection struct {
		RpcConn     *grpc.ClientConn
		RpcEndpoint string
	}

	ServiceConnection struct {
		ServiceName service.CloudrunServiceName
		Connection  *ServiceGRPCConnection
	}

	ServiceConnectionList []*ServiceConnection
)

func (c *ServiceConnection) GetAuthenticateGRPCContextFn(
	isAuthRequired bool,
) auth.AuthenticateGRPCContextFn {
	return func(ctx context.Context) (context.Context, error) {
		return c.AuthenticateGRPCContext(ctx, isAuthRequired)
	}
}

func (c *ServiceConnection) AuthenticateGRPCContext(
	ctx context.Context,
	isAuthRequired bool,
) (context.Context, error) {
	ctx, err :=
		internalAuth.AuthenticateGRPCContext(ctx, c.Connection.RpcEndpoint, isAuthRequired)
	if err != nil {
		return ctx, errors.WithStack(err)
	}

	return ctx, nil
}

func (l ServiceConnectionList) GetConnectionForService(
	s service.CloudrunServiceName,
) (*ServiceConnection, error) {
	return findDialedConnectionForService(s, l)
}

func findDialedConnectionForService(
	s service.CloudrunServiceName,
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
