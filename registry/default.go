package registry

import (
	"github.com/DMwangnima/nacos-plugin"
	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"net"
	"os"
	"strconv"
)

var (
	servers      []server
	namespace    string
	namespaceEnv = "NACOS_NAMESPACE"
	serversEnv   = []string{
		"NACOS_SERVER_1",
		"NACOS_SERVER_2",
		"NACOS_SERVER_3",
	}
)

type server struct {
	ip   string
	port int
}

func init() {
	readServers()
	readNamespace()
}

func readServers() {
	servers = make([]server, 0)
	for _, s := range serversEnv {
		addr := os.Getenv(s)
		if addr == "" {
			continue
		}
		// 做一个简单校验
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			logger.Fatal(err)
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			logger.Fatal(err)
		}
		newServer := server{
			ip:   host,
			port: portInt,
		}
		servers = append(servers, newServer)
	}
}

func readNamespace() {
	n := os.Getenv(namespaceEnv)
	if n == "" {
		logger.Fatal("missing namespace")
	}
	namespace = n
}

func NewDefaultRegistry(serviceName string) registry.Registry {
	nodes := make([]nacos.ServerNode, len(servers))
	for i, s := range servers {
		node := nacos.ServerNode{
			nacos.IpAddr(s.ip),
			nacos.Port(s.port),
		}
		nodes[i] = node
	}
	cli := nacos.Client(
		nacos.NamespaceId(namespace),
	)
	srv := nacos.Server(nodes...)
	ins := nacos.Instance(
		nacos.Weight(10),
		nacos.Enable(true),
		nacos.ServiceName(serviceName),
		nacos.Healthy(true),
		nacos.Ephemeral(true),
	)
	return NewRegistry(cli, srv, ins)
}
