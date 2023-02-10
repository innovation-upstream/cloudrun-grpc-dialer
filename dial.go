package dialer

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcMetadata "google.golang.org/grpc/metadata"
)

type (
	CloudrunServiceName string

	serviceEndpoint struct {
		ServiceName CloudrunServiceName
		RpcEndpoint string
	}

	serviceGRPCConnection struct {
		RpcConn     *grpc.ClientConn
		RpcEndpoint string
	}

	serviceConnection struct {
		ServiceName CloudrunServiceName
		Connection  *serviceGRPCConnection
	}

	serviceConnectionList []*serviceConnection

	AuthenticateGRPCContextFn func(context.Context) (context.Context, error)

	CloudrunGRPCDialer interface {
		getCloudrunEndPointForService(svc CloudrunServiceName) serviceEndpoint

		DialGRPCServices(
			ctx context.Context,
			svcs []CloudrunServiceName,
			isTLS,
			isAuthRequired bool,
		) (serviceConnectionList, func(), error)

		DialGRPCService(
			ctx context.Context,
			svcs CloudrunServiceName,
			isTLS bool,
			isAuthRequired bool,
		) (serviceConnection, func(), error)

		dialGRPC(
			ctx context.Context,
			rpcEndpoint string,
			isTLS bool,
			isAuthRequired bool,
		) (context.Context, *grpc.ClientConn, error)
	}

	getEndpointForServiceFn func(
		label CloudrunServiceName,
	) serviceEndpoint

	cloudrunGRPCDialer struct {
		cloudrunID                    string
		cloudrunRegion                string
		port                          string
		getDevEnvEndpointForServiceFn getEndpointForServiceFn
		isCloudrunEnv                 bool
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
	label CloudrunServiceName,
) serviceEndpoint {
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

	return serviceEndpoint{
		RpcEndpoint: ep,
		ServiceName: label,
	}
}

func (l *cloudrunGRPCDialer) getEndpointForServices(
	labels []CloudrunServiceName,
) []serviceEndpoint {
	if labels == nil || len(labels) == 0 {
		return nil
	}

	ix := len(labels) - 1
	head := labels[ix]
	tail := labels[0:ix]
	chunk := l.getEndpointForServices(tail)
	var endpoint serviceEndpoint
	if l.isCloudrunEnv {
		endpoint = l.getCloudrunEndPointForService(head)
	} else {
		endpoint = l.getDevEnvEndpointForServiceFn(head)
	}

	return append(chunk, endpoint)
}

func (l *cloudrunGRPCDialer) getEndpointForService(
	label CloudrunServiceName,
) serviceEndpoint {
	var endpoint serviceEndpoint
	if l.isCloudrunEnv {
		endpoint = l.getCloudrunEndPointForService(label)
	} else {
		endpoint = l.getDevEnvEndpointForServiceFn(label)
	}

	return endpoint
}

func (l *cloudrunGRPCDialer) DialGRPCServices(
	ctx context.Context,
	svcs []CloudrunServiceName,
	isTLS,
	isAuthRequired bool,
) (serviceConnectionList, func(), error) {
	svcConns, cleanup, err := l.dialGRPCServices(ctx, svcs, isTLS, isAuthRequired)
	doCleanup := func() {
		for _, c := range cleanup {
			c()
		}
	}
	if err != nil {
		return svcConns, doCleanup, errors.WithStack(err)
	}

	return svcConns, doCleanup, nil
}

func (l *cloudrunGRPCDialer) DialGRPCService(
	ctx context.Context,
	svc CloudrunServiceName,
	isTLS,
	isAuthRequired bool,
) (serviceConnection, func(), error) {
	ep := l.getEndpointForService(svc)
	var emptySvcConn serviceConnection

	conn, cleanup, err :=
		l.dialGRPCService(ctx, ep.RpcEndpoint, isTLS, isAuthRequired)
	if err != nil {
		return emptySvcConn, cleanup, errors.WithStack(err)
	}

	svcConn := serviceConnection{
		Connection:  conn,
		ServiceName: svc,
	}

	return svcConn, cleanup, nil
}

func (l *cloudrunGRPCDialer) dialGRPCServices(
	ctx context.Context,
	eps []CloudrunServiceName,
	isTLS,
	isAuthRequired bool,
) (serviceConnectionList, []func(), error) {
	emptySvcConns := serviceConnectionList{}

	if len(eps) == 0 {
		return emptySvcConns, nil, nil
	}

	head := eps[0]
	tail := eps[1:]
	chunk, cleanupChunk, err :=
		l.dialGRPCServices(ctx, tail, isTLS, isAuthRequired)
	if err != nil {
		return emptySvcConns, cleanupChunk, errors.WithStack(err)
	}

	conn, cleanup, err := l.DialGRPCService(ctx, head, isTLS, isAuthRequired)
	cleanupAll := append(cleanupChunk, cleanup)
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
) (*serviceGRPCConnection, func(), error) {
	_, c, err := l.dialGRPC(ctx, endpoint, isTLS, isAuthRequired)
	cleanup := func() {
		c.Close()
	}
	if err != nil {
		return nil, cleanup, errors.WithStack(err)
	}

	conn := &serviceGRPCConnection{
		RpcConn:     c,
		RpcEndpoint: endpoint,
	}

	return conn, cleanup, nil
}

func (l serviceConnectionList) GetConnectionForService(
	s CloudrunServiceName,
) (*serviceConnection, error) {
	return findDialedConnectionForService(s, l)
}

func findDialedConnectionForService(
	s CloudrunServiceName,
	connList serviceConnectionList,
) (*serviceConnection, error) {
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

func authenticateGRPCContext(
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

func (g *cloudrunGRPCDialer) dialGRPC(
	ctx context.Context,
	rpcEndpoint string,
	isTLS bool,
	isAuthRequired bool,
) (context.Context, *grpc.ClientConn, error) {
	ctx, err := authenticateGRPCContext(ctx, rpcEndpoint, isAuthRequired)
	if err != nil {
		return ctx, nil, err
	}

	var opts []grpc.DialOption
	if isTLS {
		creds := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	interceptor := otelgrpc.UnaryClientInterceptor()
	opts = append(opts, grpc.WithUnaryInterceptor(interceptor))
	conn, err := grpc.DialContext(ctx, rpcEndpoint, opts...)
	if err != nil {
		return ctx, conn, err
	}

	return ctx, conn, nil
}

func (c *serviceConnection) GetAuthenticateGRPCContextFn(
	isAuthRequired bool,
) AuthenticateGRPCContextFn {
	return func(ctx context.Context) (context.Context, error) {
		return c.AuthenticateGRPCContext(ctx, isAuthRequired)
	}
}

func (c *serviceConnection) AuthenticateGRPCContext(
	ctx context.Context,
	isAuthRequired bool,
) (context.Context, error) {
	ctx, err :=
		authenticateGRPCContext(ctx, c.Connection.RpcEndpoint, isAuthRequired)
	if err != nil {
		return ctx, errors.WithStack(err)
	}

	return ctx, nil
}
