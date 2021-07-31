package registry

import (
	"errors"
	"fmt"
	"github.com/DMwangnima/nacos-plugin"
	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net"
	"strconv"
	"strings"
	"sync"
)

type nacosRegistry struct {
	client   nacos.ClientOptions
	server   []nacos.ServerOptions
	instance nacos.InstanceOptions
	options  registry.Options
	naming   naming_client.INamingClient
	mu       sync.Mutex
	// set,用于记录已查询过的服务
	services map[string]struct{}
	// 这个channel应该为缓存channel，否则有阻塞的风险
	// 用于向watcher传递新请求的服务
	serviceChan chan string
	// 标记是否调用过Watch
	watchFlag bool
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	n := &nacosRegistry{
		client:      nacos.ClientOptions{*constant.NewClientConfig()},
		server:      make([]nacos.ServerOptions, 0),
		services:    make(map[string]struct{}),
		serviceChan: make(chan string, 10),
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

// 不要调用该函数，通过NewRegistry完成初始化
func (n *nacosRegistry) Init(opts ...registry.Option) error {
	return nil
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
	ip := localIP()
	if ip == "" {
		return errors.New("network failed")
	}
	n.instance.Ip = ip
	n.instance.Port = uint64(port)
	return nil
}

// Register和Deregister都只负责当前Service的注册和撤销
func (n *nacosRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	fmt.Println("reg")
	if n.naming == nil {
		return errors.New("nacos registry hasn't been initialized")
	}

	err := n.updateInstance(s)
	if err != nil {
		return err
	}
	// TODO:考虑将逻辑抽离出来
	logger.Log(logger.InfoLevel, "nacos starting register")
	_, err = n.naming.RegisterInstance(n.instance.RegisterInstanceParam)
	if err != nil {
		logger.Logf(logger.ErrorLevel, "nacos registered failed, err: %v", err)
		return err
	}
	logger.Log(logger.InfoLevel, "nacos registered successful")
	return nil
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
	logger.Log(logger.InfoLevel, "nacos starting deregister")
	_, err := n.naming.DeregisterInstance(param)
	if err != nil {
		logger.Logf(logger.ErrorLevel, "nacos deregistered failed, err: %v", err)
	}
	logger.Log(logger.InfoLevel, "nacos deregistered successful")
	return err
}

func (n *nacosRegistry) GetService(s string, opts ...registry.GetOption) ([]*registry.Service, error) {
	if n.naming == nil {
		return nil, errors.New("nacos registry hasn't been initialized")
	}

	s = divideNamespace(s)

	// TODO:考虑是否将clusters与groupName设置为和n.Instance相同的属性
	param := vo.GetServiceParam{
		Clusters:    nil,
		ServiceName: s,
		GroupName:   "",
	}

	service, err := n.naming.GetService(param)
	if err != nil {
		logger.Logf(logger.ErrorLevel, "nacos getservice failed, err:%v", err)
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
	// 更新服务set, 并将新的service推入serviceChan中
	// 思考是否放在另外一个goroutine会更好
	n.mu.Lock()
	if _, ok := n.services[s]; !ok {
		n.services[s] = struct{}{}
		// 检查是否开启了watcher
		if n.watchFlag {
			n.serviceChan <- s
		}
	}
	n.mu.Unlock()

	return []*registry.Service{rService}, nil
}

func (n *nacosRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	if n.naming == nil {
		return nil, errors.New("nacos registry hasn't been initialized")
	}

	var page uint32 = 1
	var serviceList model.ServiceList
	serviceNames := []string{}
	param := vo.GetAllServiceInfoParam{
		NameSpace: n.client.NamespaceId,
		GroupName: "",
		PageNo:    page,
		PageSize:  10000, // 一次性读完所有服务
	}
	serviceList, err := n.naming.GetAllServicesInfo(param)
	if err != nil {
		logger.Logf(logger.ErrorLevel, "nacos listServices failed, err:%v", err)
		return nil, err
	}
	serviceNames = append(serviceNames, serviceList.Doms...)

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
	n.mu.Lock()
	n.watchFlag = true
	n.mu.Unlock()
	return newNacosWatcher(n, opts...)
}

func (n *nacosRegistry) String() string {
	return "nacos"
}

func localIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil || conn == nil {
		return ""
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

// 拆分命名空间，eg: test.Stest1 返回test作为主要命名空间 test.Stest1.Stest2 返回test.Stest1作为主要命名空间
func divideNamespace(s string) string {
	ind := strings.LastIndex(s, ".")
	if ind == -1 {
		return s
	}
	return s[:ind]
}

