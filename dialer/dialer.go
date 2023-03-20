package dialer

import (
	"context"
	"strings"

	"github.com/innovation-upstream/cloudrun-grpc-dialer/internal/connection"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type (
	CloudrunGRPCDialer interface {
		getCloudrunEndPointForService(svc connection.CloudrunServiceName) connection.ServiceEndpoint

		DialGRPCServices(
			ctx context.Context,
			svcs []connection.CloudrunServiceName,
			isTLS,
			isAuthRequired bool,
			dialOpts ...grpc.DialOption,
		) (connection.ServiceConnectionList, func(), error)

		DialGRPCService(
			ctx context.Context,
			svcs connection.CloudrunServiceName,
			isTLS bool,
			isAuthRequired bool,
			dialOpts ...grpc.DialOption,
		) (connection.ServiceConnection, func(), error)

		dialGRPC(
			ctx context.Context,
			rpcEndpoint string,
			isTLS bool,
			isAuthRequired bool,
			dialOpts ...grpc.DialOption,
		) (context.Context, *grpc.ClientConn, error)
	}

	getEndpointForServiceFn func(
		label connection.CloudrunServiceName,
	) connection.ServiceEndpoint

	cloudrunGRPCDialer struct {
		cloudrunID                    string
		cloudrunRegion                string
		port                          string
		getDevEnvEndpointForServiceFn getEndpointForServiceFn
		isCloudrunEnv                 bool
		authContextFn                 func(
			ctx context.Context,
			addr string,
			isAuthRequired bool,
		) (context.Context, error)
		dialFn func(
			ctx context.Context,
			target string,
			opts ...grpc.DialOption,
		) (conn *grpc.ClientConn, err error)
		getSecureDialOptionsFn   func() []grpc.DialOption
		getInsecureDialOptionsFn func() []grpc.DialOption
	}

	// CloudrunGRPCDialerFactory instantiates and returns a new
	// CloudrunServiceDialer
	// cloudRunID - Your project's Cloudrun ID
	// cloudRunID - Your project's Cloudrun ID
	CloudrunGRPCDialerFactory func(
		cloudRunID string,
		cloudRunRegion string,
		opts ...cloudrunGRPCDialerOption,
	) CloudrunGRPCDialer
)

var NewCloudrunGRPCDialer CloudrunGRPCDialerFactory = func(
	cloudrunID string,
	cloudrunRegion string,
	opts ...cloudrunGRPCDialerOption,
) CloudrunGRPCDialer {
	d := &cloudrunGRPCDialer{
		cloudrunID:     cloudrunID,
		cloudrunRegion: cloudrunRegion,
	}
	withDefaultOpts := d.applyOptions(
		withDefaultPort(),
		withDefaultGetDevEnvEndPointForServiceFn(),
		withDefaultIsCloudrunEnv(),
		withDefaultAuthContextFn(),
		withDefaultDialFn(),
		withDefaultGetSecureDialOptionsFn(),
		withDefaultGetInsecureDialOptionsFn(),
	)
	withOpts := withDefaultOpts.applyOptions(opts...)

	return withOpts
}

func (l *cloudrunGRPCDialer) applyOptions(
	opts ...cloudrunGRPCDialerOption,
) *cloudrunGRPCDialer {
	if opts == nil || len(opts) == 0 {
		return l
	}

	head := opts[0]
	tail := opts[1:]
	chunk := l.applyOptions(tail...)

	return head(chunk)
}

func (l *cloudrunGRPCDialer) getCloudrunEndPointForService(
	label connection.CloudrunServiceName,
) connection.ServiceEndpoint {
	var sb strings.Builder
	sb.WriteString(string(label))
	sb.WriteRune('-')
	sb.WriteString(l.cloudrunID)
	sb.WriteRune('-')
	sb.WriteString(l.cloudrunRegion)
	sb.WriteString(".run.app")
	sb.WriteRune(':')
	sb.WriteString(l.port)
	ep := sb.String()

	return connection.ServiceEndpoint{
		RpcEndpoint: ep,
		ServiceName: label,
	}
}

func (l *cloudrunGRPCDialer) getEndpointForServices(
	labels []connection.CloudrunServiceName,
) []connection.ServiceEndpoint {
	if labels == nil || len(labels) == 0 {
		return nil
	}

	ix := len(labels) - 1
	head := labels[ix]
	tail := labels[0:ix]
	chunk := l.getEndpointForServices(tail)
	var endpoint connection.ServiceEndpoint
	if l.isCloudrunEnv {
		endpoint = l.getCloudrunEndPointForService(head)
	} else {
		endpoint = l.getDevEnvEndpointForServiceFn(head)
	}

	return append(chunk, endpoint)
}

func (l *cloudrunGRPCDialer) getEndpointForService(
	label connection.CloudrunServiceName,
) connection.ServiceEndpoint {
	var endpoint connection.ServiceEndpoint
	if l.isCloudrunEnv {
		endpoint = l.getCloudrunEndPointForService(label)
	} else {
		endpoint = l.getDevEnvEndpointForServiceFn(label)
	}

	return endpoint
}

func (l *cloudrunGRPCDialer) DialGRPCService(
	ctx context.Context,
	svc connection.CloudrunServiceName,
	isTLS,
	isAuthRequired bool,
	dialOpts ...grpc.DialOption,
) (connection.ServiceConnection, func(), error) {
	ep := l.getEndpointForService(svc)
	var emptySvcConn connection.ServiceConnection

	conn, cleanup, err :=
		l.dialGRPCService(ctx, ep.RpcEndpoint, isTLS, isAuthRequired, dialOpts...)
	if err != nil {
		return emptySvcConn, cleanup, errors.WithStack(err)
	}

	svcConn := connection.ServiceConnection{
		Connection:  conn,
		ServiceName: svc,
	}

	return svcConn, cleanup, nil
}

func (l *cloudrunGRPCDialer) DialGRPCServices(
	ctx context.Context,
	eps []connection.CloudrunServiceName,
	isTLS,
	isAuthRequired bool,
	dialOpts ...grpc.DialOption,
) (connection.ServiceConnectionList, func(), error) {
	emptySvcConns := connection.ServiceConnectionList{}

	if len(eps) == 0 {
		return emptySvcConns, nil, nil
	}

	head := eps[0]
	tail := eps[1:]
	chunk, cleanupChunk, err :=
		l.DialGRPCServices(ctx, tail, isTLS, isAuthRequired, dialOpts...)
	if err != nil {
		return emptySvcConns, cleanupChunk, errors.WithStack(err)
	}

	conn, cleanup, err := l.DialGRPCService(ctx, head, isTLS, isAuthRequired, dialOpts...)
	cleanupAll := func() {
		cleanupChunk()
		cleanup()
	}
	if err != nil {
		return emptySvcConns, cleanupChunk, errors.WithStack(err)
	}

	return append(chunk, &conn), cleanupAll, nil
}

func (l *cloudrunGRPCDialer) dialGRPCService(
	ctx context.Context,
	endpoint string,
	isTLS bool,
	isAuthRequired bool,
	dialOpts ...grpc.DialOption,
) (*connection.ServiceGRPCConnection, func(), error) {
	_, c, err := l.dialGRPC(ctx, endpoint, isTLS, isAuthRequired, dialOpts...)
	cleanup := func() {
		c.Close()
	}
	if err != nil {
		return nil, cleanup, errors.WithStack(err)
	}

	conn := &connection.ServiceGRPCConnection{
		RpcConn:     c,
		RpcEndpoint: endpoint,
	}

	return conn, cleanup, nil
}

func (g *cloudrunGRPCDialer) dialGRPC(
	ctx context.Context,
	rpcEndpoint string,
	isTLS bool,
	isAuthRequired bool,
	dialOpts ...grpc.DialOption,
) (context.Context, *grpc.ClientConn, error) {
	ctx, err := g.authContextFn(ctx, rpcEndpoint, isAuthRequired)
	if err != nil {
		return ctx, nil, err
	}

	if isTLS {
		tlsOpt := g.getSecureDialOptionsFn()
		dialOpts = append(dialOpts, tlsOpt...)
	} else {
		noTlsOpt := g.getInsecureDialOptionsFn()
		dialOpts = append(dialOpts, noTlsOpt...)
	}

	conn, err := g.dialFn(ctx, rpcEndpoint, dialOpts...)
	if err != nil {
		return ctx, conn, err
	}

	return ctx, conn, nil
}
