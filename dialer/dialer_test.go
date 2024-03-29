package dialer

import (
	"context"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/innovation-upstream/cloudrun-grpc-dialer/internal/connection"
	"github.com/innovation-upstream/cloudrun-grpc-dialer/service"
	"google.golang.org/grpc"
)

const (
	cloudrunID     = "testID"
	cloudrunRegion = "testRegion"
	port           = "testPort"
)

// Generated By GPT3
func TestCloudrunGRPCDialerGetCloudrunEndPointForService(c *testing.T) {
	err := quick.Check(func(label service.CloudrunServiceName) bool {
		l := &cloudrunGRPCDialer{
			cloudrunID:     cloudrunID,
			cloudrunRegion: cloudrunRegion,
			port:           port,
		}
		ep := l.getCloudrunEndPointForService(label)
		expect := string(label) + "-" + l.cloudrunID + "-" + l.cloudrunRegion + ".run.app:" + l.port

		return ep.RpcEndpoint == expect
	}, nil)
	if err != nil {
		c.Error(err)
	}
}

func TestGetEndpointForService(t *testing.T) {
	f := func(label service.CloudrunServiceName, isCloudrunEnv bool) bool {
		opts := []cloudrunGRPCDialerOption{
			func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
				d.isCloudrunEnv = isCloudrunEnv
				return d
			},
		}

		l := NewCloudrunGRPCDialer(cloudrunID, cloudrunRegion, opts...).(*cloudrunGRPCDialer)
		endpoint := l.getEndpointForService(label)
		if isCloudrunEnv {
			return endpoint == l.getCloudrunEndPointForService(label)
		} else {
			return endpoint == l.getDevEnvEndpointForServiceFn(label)
		}
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestDialGRPC(t *testing.T) {
	f := func(
		inputRpcEndpoint string,
		inputIsTLS bool,
		inputIsAuthRequired bool,
	) bool {
		// create mock objects
		var authContextFnCalled, dialFnCalled, getSecureDialOptionsFnCalled, getInsecureDialOptionsFnCalled bool

		mockAuthContextFn := func(ctx context.Context, rpcEndpoint string, isAuthRequired bool) (context.Context, error) {
			authContextFnCalled = true
			return ctx, nil
		}

		mockDialFn := func(ctx context.Context, rpcEndpoint string, dialOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
			dialFnCalled = true
			return nil, nil
		}

		mockGetSecureDialOptionsFn := func() []grpc.DialOption {
			getSecureDialOptionsFnCalled = true
			return []grpc.DialOption{}
		}

		mockGetInsecureDialOptionsFn := func() []grpc.DialOption {
			getInsecureDialOptionsFnCalled = true
			return []grpc.DialOption{}
		}

		inputCtx := context.TODO()

		g := NewCloudrunGRPCDialer(
			"",
			"",
			func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
				d.authContextFn = mockAuthContextFn
				d.dialFn = mockDialFn
				d.getSecureDialOptionsFn = mockGetSecureDialOptionsFn
				d.getInsecureDialOptionsFn = mockGetInsecureDialOptionsFn
				return d
			},
		)

		// call function to test
		outCtx, outConn, err := g.dialGRPC(
			inputCtx,
			inputRpcEndpoint,
			inputIsTLS,
			inputIsAuthRequired,
		)

		a1 := reflect.DeepEqual(inputCtx, outCtx)
		a2 := outConn == nil
		a3 := err == nil

		var correctTlsFnCalled bool
		if inputIsTLS {
			correctTlsFnCalled = getSecureDialOptionsFnCalled
		} else {
			correctTlsFnCalled = getInsecureDialOptionsFnCalled
		}

		return a1 && a2 && a3 && dialFnCalled && authContextFnCalled && correctTlsFnCalled
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

type (
	TestDialer interface {
	}
	testDialer struct {
		realStruct *cloudrunGRPCDialer
	}
)

func (l *testDialer) DialGRPCServices(
	ctx context.Context,
	eps []service.CloudrunServiceName,
	isTLS,
	isAuthRequired bool,
	dialOpts ...grpc.DialOption,
) (connection.ServiceConnectionList, func(), error) {
	return l.realStruct.DialGRPCServices(
		ctx, eps, isTLS, isAuthRequired, dialOpts...,
	)
}

func (l *testDialer) DialGRPCService(
	ctx context.Context,
	svc service.CloudrunServiceName,
	isTLS,
	isAuthRequired bool,
	dialOpts ...grpc.DialOption,
) (connection.ServiceConnection, func(), error) {
	return connection.ServiceConnection{
		ServiceName: svc,
		Connection:  nil,
	}, nil, nil
}
