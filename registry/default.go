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
	servers          []server
	namespace        string
	num              int
	namespaceEnv     = "NACOS_NAMESPACE"
	serverNumEnv     = "NACOS_SERVER_NUM"
	serversEnvPrefix = "NACOS_SERVER_"
)

type server struct {
	ip   string
	port int
}

func readNum() {
	var err error
	n := os.Getenv(serverNumEnv)
	num, err = strconv.Atoi(n)
	if err != nil {
		logger.Fatalf("nacos server num is wrong, err: %s", err)
	}
	if num <= 0 {
		logger.Fatalf("nacos server num is invalid, num: %d", num)
	}
}

func readServers() {
	for i := 1; i <= num; i++ {
		env := serversEnvPrefix + strconv.Itoa(i)
		addr := os.Getenv(env)
		if addr == "" {
			logger.Fatal("missing nacos server address")
		}
		// 做一个简单校验
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			logger.Fatalf("nacos server address is wrong, err: %s", err)
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			logger.Fatalf("nacos server port is wrong, err: %s", err)
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
		logger.Fatal("missing nacos server namespace")
	}
	namespace = n
}

func NewDefaultRegistry(serviceName string, ephemeral bool) registry.Registry {
	readNum()
	readServers()
	readNamespace()
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
		nacos.Ephemeral(ephemeral),
	)
	return NewRegistry(cli, srv, ins)
}
