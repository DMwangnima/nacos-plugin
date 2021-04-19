package registry

import (
	"github.com/asim/go-micro/v3"
	"testing"
)

func TestNewDefaultRegistry(t *testing.T) {
	reg := NewDefaultRegistry("helloworld")
	serv := micro.NewService(
		micro.Name("helloworld"),
		micro.Registry(reg),
	)
	serv.Run()
}
