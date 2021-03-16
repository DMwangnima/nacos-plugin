package nacos

import (
	"errors"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net"
	"strconv"
)

type nacosRegistry struct {
	client   ClientOptions
	server   []ServerOptions
	instance InstanceOptions
	options  registry.Options
	naming   naming_client.INamingClient
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	n := &nacosRegistry{
		client: ClientOptions{*constant.NewClientConfig()},
		server: make([]ServerOptions, 0),
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

	// 初始化client
	if cliOpts, ok := n.options.Context.Value(clientKey{}).([]ClientOption); ok {
		for _, cliOpt := range cliOpts {
			cliOpt(&n.client)
		}
		if n.client.NamespaceId == "" {
			return errors.New("missing namespace of client")
		}
	} else {
		return errors.New("missing client options")
	}

	// 初始化server
	var defIp = "127.0.0.1"
	var defPort uint64 = 8848
	if nodes, ok := n.options.Context.Value(serverKey{}).([]ServerNode); ok {
		for _, node := range nodes {
			srvOptions := ServerOptions{*constant.NewServerConfig(defIp, defPort)}
			for _, opt := range node {
				opt(&srvOptions)
			}
			if srvOptions.IpAddr == defIp {
				return errors.New("missing ipAddr of nacos server")
			}
			n.server = append(n.server, srvOptions)
		}
	} else {
		return errors.New("missing server options")
	}

	// 初始化instance
	if insOpts, ok := n.options.Context.Value(instanceKey{}).([]InstanceOption); ok {
		for _, insOpt := range insOpts {
			insOpt(&n.instance)
		}
	} else {
		return errors.New("missing instance options")
	}

	// 生成namingClient
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

// 逻辑暂定为只注册一个节点
func (n *nacosRegistry) updateInstance(s *registry.Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("require service owning at least one node")
	}
	node := s.Nodes[0]
	_, portStr, err := net.SplitHostPort(node.Address)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}
	n.instance.Port = uint64(port)
	return nil
}

// Register和Deregister都只负责当前Service的注册和撤销
func (n *nacosRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if n.naming == nil {
		return errors.New("nacos registry hasn't been initialized")
	}

	err := n.updateInstance(s)
	if err != nil {
		return err
	}
	_, err = n.naming.RegisterInstance(n.instance.RegisterInstanceParam)
	return err
}

func (n *nacosRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if n.naming == nil {
		return errors.New("nacos registry hasn't been initialized")
	}
	param := vo.DeregisterInstanceParam{
		Ip:          n.instance.Ip,
		Port:        n.instance.Port,
		Cluster:     n.instance.ClusterName,
		ServiceName: n.instance.ServiceName,
		GroupName:   n.instance.GroupName,
		Ephemeral:   n.instance.Ephemeral,
	}
	_, err := n.naming.DeregisterInstance(param)
	return err
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
