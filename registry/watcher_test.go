package registry

import (
	"fmt"
	"github.com/DMwangnima/nacos-plugin"
	"github.com/asim/go-micro/v3/registry"
	"sync"
	"testing"
)

func newRegistry() registry.Registry {
	cli := nacos.Client(
		nacos.NamespaceId("a9c64542-87bf-432e-816f-4bd5c19806ac"),
	)
	node := nacos.ServerNode{
		nacos.IpAddr("192.168.3.7"),
		nacos.Port(8848),
	}
	node1 := nacos.ServerNode{
		nacos.IpAddr("192.168.3.15"),
		nacos.Port(8848),
	}
	node2 := nacos.ServerNode{
		nacos.IpAddr("192.168.3.16"),
		nacos.Port(8848),
	}
	srv := nacos.Server(node, node1, node2)
	ins := nacos.Instance(
		nacos.Weight(10),
		nacos.Enable(true),
		nacos.ServiceName("nacos-watcher"),
		nacos.Healthy(true),
		nacos.Ephemeral(true),
	)
	reg := NewRegistry(cli, srv, ins)
	return reg
}

func TestNacosWatcher_Next(t *testing.T) {
	reg := newRegistry()
	w, err := reg.Watch()
	if err != nil {
		fmt.Println(err)
		return
	}
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			reg.GetService("helloworldserver")
			reg.GetService("helloworldclient")
			wg.Done()
		}()
	}
	wg.Wait()
	for i := 0; i < 10; i++ {
		res, err := w.Next()
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, node := range res.Service.Nodes {
			fmt.Println(node.Address)
		}
	}
}
