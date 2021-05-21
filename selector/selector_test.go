package selector

import (
	"fmt"
	"github.com/DMwangnima/nacos-plugin"
	nacosReg "github.com/DMwangnima/nacos-plugin/registry"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/selector"
	"testing"
	"time"
)

const (
	SERVICE_NAME = "selector"
)

func newRegistry() registry.Registry {
	cli := nacos.Client(
		nacos.NamespaceId("415c7259-d269-47e3-9e83-ccb8abbb0247"),
	)
	node := nacos.ServerNode{
		nacos.IpAddr("192.168.26.182"),
		nacos.Port(8848),
	}
	node1 := nacos.ServerNode{
		nacos.IpAddr("192.168.26.183"),
		nacos.Port(8848),
	}
	node2 := nacos.ServerNode{
		nacos.IpAddr("192.168.26.184"),
		nacos.Port(8848),
	}
	srv := nacos.Server(node, node1, node2)
	ins := nacos.Instance(
		nacos.Ephemeral(true),
		nacos.Healthy(true),
		nacos.ServiceName("gateway"),
		nacos.Weight(10),
		nacos.Enable(true),
	)
	return nacosReg.NewRegistry(cli, srv, ins)
}

func printErr(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}

func TestRepeatSelect(t *testing.T) {
	reg := newRegistry()
	s := NewSelector(selector.Registry(reg))
	service := "broker_client1"
	for i := 0; i < 1000; i++ {
		next, _ := s.Select(service)
		node, err := next()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(node.Id)
	}
}

func TestDynamicTest(t *testing.T) {
	reg := newRegistry()
	s := NewSelector(selector.Registry(reg))
	service := "helloworldserver"
	next, err := s.Select(service)
	printErr(err)
	for i := 0; i < 10; i++ {
		node, err := next()
		printErr(err)
		fmt.Println(node.Address)
	}
	time.Sleep(120 * time.Second)
	for i := 0; i < 10; i++ {
		node, err := next()
		printErr(err)
		fmt.Println(node.Address)
	}
}

func TestSelect(t *testing.T) {
	reg := newRegistry()
	s := NewSelector(selector.Registry(reg))
	service := "helloworldserver"
	next, err := s.Select(service)
	finish := make(chan struct{})
	go func() {
		next, err := s.Select("helloworldclient")
		printErr(err)
		for i := 0; i < 10000; i++ {
			node, err := next()
			printErr(err)
			fmt.Println("client:" + node.Address)
		}
		finish <- struct{}{}
	}()
	printErr(err)
	for i := 0; i < 10000; i++ {
		node, err := next()
		printErr(err)
		fmt.Println("server:" + node.Address)
	}
	<-finish
}

func TestMark(t *testing.T) {
	reg := newRegistry()
	s := NewSelector(selector.Registry(reg))

	service := "helloworldserver"
	next, err := s.Select(service)
	printErr(err)
	node, err := next()
	if node == nil {
		return
	}
	fmt.Println(node.Address)
	s.Mark(service, node, nil)
	node, err = next()
	printErr(err)
	fmt.Println(node.Address)
	aNext, err := s.Select(service)
	node, err = aNext()
	printErr(err)
	fmt.Println(node.Address)
}
