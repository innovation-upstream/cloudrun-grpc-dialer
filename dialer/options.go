package dialer

import (
	"crypto/tls"
	"os"

	"github.com/innovation-upstream/cloudrun-grpc-dialer/internal/auth"
	"github.com/innovation-upstream/cloudrun-grpc-dialer/internal/connection"
	"github.com/innovation-upstream/cloudrun-grpc-dialer/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type (
	cloudrunGRPCDialerOption func(*cloudrunGRPCDialer) *cloudrunGRPCDialer
)

func WithPort(port string) cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.port = port
		return d
	}
}

func getDefaultPort() string {
	envPort := os.Getenv("PORT")
	if envPort != "" {
		return envPort
	}

	return "443"
}

func withDefaultPort() cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.port = getDefaultPort()
		return d
	}
}

func getDefaultGetDevEnvEndPointForServiceFn() getEndpointForServiceFn {
	return func(n service.CloudrunServiceName) connection.ServiceEndpoint {
		return connection.ServiceEndpoint{
			ServiceName: n,
			RpcEndpoint: "service:443",
		}
	}
}

func withDefaultGetDevEnvEndPointForServiceFn() cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.getDevEnvEndpointForServiceFn = getDefaultGetDevEnvEndPointForServiceFn()
		return d
	}
}

func withDefaultIsCloudrunEnv() cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.isCloudrunEnv = os.Getenv("ENVIRONMENT") == "production"
		return d
	}
}

func withDefaultAuthContextFn() cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.authContextFn = auth.AuthenticateGRPCContext
		return d
	}
}

func withDefaultDialFn() cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.dialFn = grpc.DialContext
		return d
	}
}

func withDefaultGetSecureDialOptionsFn() cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.getSecureDialOptionsFn = func() []grpc.DialOption {
			creds := credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			})

			opt := grpc.WithTransportCredentials(creds)
			return []grpc.DialOption{opt}
		}
		return d
	}
}

func withDefaultGetInsecureDialOptionsFn() cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.getInsecureDialOptionsFn = func() []grpc.DialOption {
			opt := grpc.WithInsecure()
			return []grpc.DialOption{opt}
		}
		return d
	}
}

func withDefaultDefaultDialOptions() cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.defaultDialOptions = make([]grpc.DialOption, 0)
		return d
	}
}

func WithDefaultDialOptions(opts ...grpc.DialOption) cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.defaultDialOptions = opts
		return d
	}
}

func WithGetDevEnvEndPointForServiceFn(
	fn getEndpointForServiceFn,
) cloudrunGRPCDialerOption {
	return func(d *cloudrunGRPCDialer) *cloudrunGRPCDialer {
		d.getDevEnvEndpointForServiceFn = fn
		return d
	}
}
