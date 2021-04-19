package registry

import (
	"errors"
	"github.com/DMwangnima/nacos-plugin"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net"
	"strconv"
)

type nacosRegistry struct {
	client   nacos.ClientOptions
	server   []nacos.ServerOptions
	instance nacos.InstanceOptions
	options  registry.Options
	naming   naming_client.INamingClient
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	n := &nacosRegistry{
		client: nacos.ClientOptions{*constant.NewClientConfig()},
		server: make([]nacos.ServerOptions, 0),
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
	if cliOpts, ok := n.options.Context.Value(nacos.ClientKey{}).([]nacos.ClientOption); ok {
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
	if nodes, ok := n.options.Context.Value(nacos.ServerKey{}).([]nacos.ServerNode); ok {
		for _, node := range nodes {
			srvOptions := nacos.ServerOptions{*constant.NewServerConfig(defIp, defPort)}
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
	if insOpts, ok := n.options.Context.Value(nacos.InstanceKey{}).([]nacos.InstanceOption); ok {
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
	n.instance.Ip = localIP()
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
			Address:  host.Ip + ":" + strconv.Itoa(int(host.Port)),
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

func (n *nacosRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	if n.naming == nil {
		return nil, errors.New("nacos registry hasn't been initialized")
	}

	var page uint32
	var serviceList model.ServiceList
	serviceNames := []string{}
	param := vo.GetAllServiceInfoParam{
		NameSpace: n.client.NamespaceId,
		GroupName: "",
		PageNo:    page,
		PageSize:  10,
	}
	get := func() {
		serviceList, _ = n.naming.GetAllServicesInfo(param)
	}
	for get(); serviceList.Count > 0; get() {
		param.PageNo += 1
		serviceNames = append(serviceNames, serviceList.Doms...)
	}

	services := []*registry.Service{}
	for _, name := range serviceNames {
		if tmpServices, err := n.GetService(name); err != nil {
			return nil, err
		} else {
			services = append(services, tmpServices...)
		}
	}

	return services, nil
}

func (n *nacosRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newNacosWatcher(n, opts...)
}

func (n *nacosRegistry) String() string {
	return "nacos"
}

func localIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}
