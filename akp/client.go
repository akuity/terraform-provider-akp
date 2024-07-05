package akp

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/akuity/grpc-gateway-client/pkg/grpc/gateway"
	"github.com/akuity/grpc-gateway-client/pkg/http/roundtripper"
)

var Marshaller = &runtime.JSONPb{
	MarshalOptions:   MarshalOptions,
	UnmarshalOptions: UnmarshalOptions,
}

var (
	MarshalOptions = protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	UnmarshalOptions = protojson.UnmarshalOptions{
		// Set DiscardUnknown as false to return error
		// if the request contains unknown fields.
		DiscardUnknown: true,
	}
)

func newClient(baseURL string, skipTLSVerify bool) gateway.Client {
	hc := &http.Client{}
	roundtripper.ApplyAuthorizationHeaderInjector(hc)
	return gateway.NewClient(baseURL,
		gateway.WithHTTPClient(hc),
		gateway.WithMarshaller(Marshaller),
		gateway.SkipTLSVerify(skipTLSVerify),
	)
}
