package nacos_plugin

import (
	"errors"
	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net"
	"strconv"
)

func init() {

}

type nacosRegistry struct {
	client  ClientOptions
	server  []ServerOptions
	options registry.Options
	naming  naming_client.INamingClient
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	n := &nacosRegistry{
		client:  ClientOptions{*constant.NewClientConfig()},
		server:  make([]ServerOptions, 0),
		options: registry.Options{},
	}
	if err := configure(n, opts...); err != nil {
		panic(err)
	}
	return n
}

func configure(n *nacosRegistry, opts ...registry.Option) error {
	for _, opt := range opts {
		opt(&n.options)
	}

	if cliOpts, ok := n.options.Context.Value(clientKey{}).([]ClientOption); ok {
		for _, cliOpt := range cliOpts {
			cliOpt(&n.client)
		}
	}

	var defIp = "127.0.0.1"
	var defPort uint64 = 80
	if nodes, ok := n.options.Context.Value(serverKey{}).([]ServerNode); ok {
		for _, node := range nodes {
			srvOptions := ServerOptions{*constant.NewServerConfig(defIp, defPort)}
			for _, opt := range node {
				opt(&srvOptions)
			}
			n.server = append(n.server, srvOptions)
		}
	}

	serverConfigs := make([]constant.ServerConfig, 0)
	for _, s := range n.server {
		serverConfigs = append(serverConfigs, s.ServerConfig)
	}
	var err error
	n.naming, err = clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &n.client.ClientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	return err
}

func (n *nacosRegistry) Init(opts ...registry.Option) error {
	return configure(n, opts...)
}

func (n *nacosRegistry) Options() registry.Options {
	return n.options
}

func newRegisterInstanceParams(s *registry.Service) (params []*vo.RegisterInstanceParam, err error) {
	if len(s.Nodes) == 0 {
		return nil, errors.New("require at least one node")
	}
	params = make([]*vo.RegisterInstanceParam, 0)
	var ferr error
	for _, node := range s.Nodes {
		_, portStr, err := net.SplitHostPort(node.Address)
		if err != nil {
			ferr = err
			continue
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			ferr = err
			continue
		}
		// 暂定
		param := &vo.RegisterInstanceParam{
			Ip:          string(localIP()),
			Port:        uint64(port),
			Weight:      1,
			Enable:      true,
			Healthy:     true,
			Metadata:    nil,
			ClusterName: "CLUSTER",
			ServiceName: s.Name,
			GroupName:   "GROUP",
			Ephemeral:   true,
		}
		params = append(params, param)
	}
	if ferr != nil {
		logger.Fatal("some nodes information are not correct")
	}
	if len(params) == 0 {
		return nil, errors.New("all the nodes are invalid")
	}
	return params, nil
}

// 考虑加入instance参数
func (n *nacosRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if n.naming == nil {
		return errors.New("nacos registry hasn't been initialized")
	}

	params, err := newRegisterInstanceParams(s)
	if err != nil {
		return err
	}
	var ferr error
	var count int
	for _, param := range params {
		_, err := n.naming.RegisterInstance(*param)
		if err != nil {
			ferr = err
			continue
		}
		count++
	}
	if count > 0 && ferr != nil {
		return errors.New("some nodes failed to register")
	}
	if count == 0 {
		return errors.New("all nodes failed to register")
	}
	return nil
}

func newDeregisterInstanceParams(s *registry.Service) (params []*vo.DeregisterInstanceParam, err error) {
	if len(s.Nodes) == 0 {
		return nil, errors.New("")
	}
	params = make([]*vo.DeregisterInstanceParam, 0)
	var ferr error
	for _, node := range s.Nodes {
		_, portStr, err := net.SplitHostPort(node.Address)
		if err != nil {
			ferr = err
			continue
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			ferr = err
			continue
		}
		// 暂定
		param := &vo.DeregisterInstanceParam{
			Ip:          string(localIP()),
			Port:        uint64(port),
			Cluster:     "CLUSTER",
			ServiceName: s.Name,
			GroupName:   "GROUP",
			Ephemeral:   true,
		}
		params = append(params, param)
	}
	if ferr != nil {
		logger.Fatal("some nodes information are not correct")
	}
	if len(params) == 0 {
		return nil, errors.New("all the nodes are invalid")
	}
	return params, nil
}

func (n *nacosRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if n.naming == nil {
		return errors.New("nacos registry hasn't been initialized")
	}

	params, err := newDeregisterInstanceParams(s)
	if err != nil {
		return err
	}
	var ferr error
	var count int
	for _, param := range params {
		_, err := n.naming.DeregisterInstance(*param)
		if err != nil {
			ferr = err
			continue
		}
		count++
	}
	if count > 0 && ferr != nil {
		return errors.New("some nodes failed to deregister")
	}
	if count == 0 {
		return errors.New("all nodes failed to deregister")
	}
	return nil
}

func (n *nacosRegistry) GetService(s string, opts ...registry.GetOption) ([]*registry.Service, error) {
	if n.naming == nil {
		return nil, errors.New("nacos registry hasn't been initialized")
	}

	param := vo.GetServiceParam{
		Clusters:    nil,
		ServiceName: s,
		GroupName:   "",
	}

	service, err := n.naming.GetService(param)
	if err != nil {
		return nil, err
	}

	nodes := make([]*registry.Node, 0, len(service.Hosts))
	for _, host := range service.Hosts {
		node := &registry.Node{
			Id:       host.InstanceId,
			Address:  host.Ip + strconv.Itoa(int(host.Port)),
			Metadata: host.Metadata,
		}
		nodes = append(nodes, node)
	}

	rService := &registry.Service{
		Name:      service.Name,
		Version:   "",
		Metadata:  service.Metadata,
		Endpoints: nil,
		Nodes:     nodes,
	}

	return []*registry.Service{rService}, nil
}

// 需要搞清楚nacos的API
func (n *nacosRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	if n.naming == nil {
		return nil, errors.New("nacos registry hasn't been initialized")
	}

	// 暂定
	param := vo.GetAllServiceInfoParam{
		NameSpace: n.client.NamespaceId,
		GroupName: "",
		PageNo:    0,
		PageSize:  0,
	}

	_, err := n.naming.GetAllServicesInfo(param)
	if err != nil {
		return nil, err
	}
	return nil, err
}

// nacos-sdk-go已经在内部实现了PushReceiver，所以并不需要再实现Watcher
func (n *nacosRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return nil, nil
}

func (n *nacosRegistry) String() string {
	return "nacos"
}

func localIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}
