package registry

import (
	"github.com/asim/go-micro/v3"
	"testing"
)

func TestNewDefaultRegistryWithMetaData(t *testing.T) {
	md := make(map[string]string)
	md["version"] = "test"
	reg := NewDefaultRegistry(
		WithServiceName("helloworld"),
		WithEphemeral(true),
		WithMetaData(md),
	)
	serv := micro.NewService(
		micro.Name("helloworld"),
		micro.Registry(reg),
	)
	serv.Run()
}
