package registry

import (
	"flag"
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
	version          string
	defaultVersion   = "default"
	versionKey       = "version"
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

func readVersion() {
	v := flag.String("version", defaultVersion, "service version")
	version = *v
}

func NewDefaultRegistry(opts ...Option) registry.Registry {
	o := options{}
	o.ephemeral = true
	for _, opt := range opts {
		opt(&o)
	}
	if o.serviceName == "" {
		logger.Fatalf("missing service name")
	}
	if o.metaData == nil {
		o.metaData = make(map[string]string)
	}

	readNum()
	readServers()
	readNamespace()
	readVersion()
	configureVersion(&o)

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
		nacos.ServiceName(o.serviceName),
		nacos.Healthy(true),
		nacos.Ephemeral(o.ephemeral),
		nacos.MetaData(o.metaData),
	)
	return NewRegistry(cli, srv, ins)
}

func configureVersion(o *options) {
	if version != defaultVersion || o.metaData[versionKey] == "" {
		o.metaData[versionKey] = version
	}
}