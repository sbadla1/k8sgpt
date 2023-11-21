package server

import (
	rpc "github.com/sbadla1/k8sgpt-schemas/protobuf/proto/go-server/schema/v1"
)

type handler struct {
	rpc.UnimplementedServerServiceServer
}
