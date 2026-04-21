// Package apiclient is a fixture for the check-api-client-fields tests.
// It mimics the generated api-client-go protobuf Go code: snake_case JSON
// tags, different initialism (ClientId vs ClientID), plus unexported
// protobuf housekeeping fields that must be ignored by the parser.
package apiclient

type protoimplMessageState struct{}
type protoimplSizeCache int32
type protoimplUnknownFields []byte

type FooSpec struct {
	state         protoimplMessageState
	sizeCache     protoimplSizeCache
	unknownFields protoimplUnknownFields

	Name      string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	ClientId  string `protobuf:"bytes,2,opt,name=client_id" json:"client_id,omitempty"`
	IssuerUrl string `protobuf:"bytes,3,opt,name=issuer_url" json:"issuer_url,omitempty"`
	Blob      []byte `protobuf:"bytes,4,opt,name=blob" json:"blob,omitempty"`
	Enabled   bool   `protobuf:"varint,5,opt,name=enabled" json:"enabled,omitempty"`
}
