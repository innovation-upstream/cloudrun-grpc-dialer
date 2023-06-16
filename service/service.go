package service

type (
	CloudrunServiceName string

	ServiceEndpoint struct {
		ServiceName CloudrunServiceName
		RpcEndpoint string
	}
)
