package dialer

import "os"

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
	return func(n CloudrunServiceName) serviceEndpoint {
		return serviceEndpoint{
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
