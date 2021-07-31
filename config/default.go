package config

import (
	"github.com/DMwangnima/nacos-plugin"
	"github.com/asim/go-micro/v3/config/source"
	"github.com/asim/go-micro/v3/logger"
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

func readServers() {
	for _, s := range serversEnv {
		addr := os.Getenv(s)
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

func NewDefaultSource(serviceName string) source.Source {
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
	cli := nacos.ConfClient(
		nacos.NamespaceId(namespace),
	)
	srv := nacos.ConfServer(nodes...)
	private := nacos.ConfParam(
		nacos.Group("DEFAULT_GROUP"),
		nacos.DataId(serviceName),
	)
	return NewSource(cli, srv, private)
}
