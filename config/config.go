package config

import (
	"errors"
	"github.com/DMwangnima/nacos-plugin"
	"github.com/asim/go-micro/v3/config/source"
	"github.com/asim/go-micro/v3/logger"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"time"
)

type nacosSource struct {
	client  nacos.ClientOptions
	server  []nacos.ServerOptions
	config  config_client.IConfigClient
	options source.Options
	param   nacos.ConfigOptions
}

func NewSource(opts ...source.Option) source.Source {
	n := &nacosSource{
		client:  nacos.ClientOptions{*constant.NewClientConfig()},
		server:  make([]nacos.ServerOptions, 0),
		options: source.NewOptions(opts...),
	}
	if err := configure(n, opts...); err != nil {
		panic(err)
	}
	return n
}

func configure(n *nacosSource, opts ...source.Option) error {
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

	serverConfigs := make([]constant.ServerConfig, 0)
	for _, s := range n.server {
		serverConfigs = append(serverConfigs, s.ServerConfig)
	}

	if param, ok := n.options.Context.Value(nacos.ConfParamKey{}).([]nacos.ConfigOption); ok {
		for _, confOpt := range param {
			confOpt(&n.param)
		}
	} else {
		return errors.New("missing confParam options")
	}

	var err error
	n.config, err = clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &n.client.ClientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	return err
}

func (n *nacosSource) Read() (*source.ChangeSet, error) {
	if n.config == nil {
		return nil, errors.New("nacos config hasn't been initialized")
	}
	content, err := n.config.GetConfig(n.param.ConfigParam)
	if err != nil {
		logger.Logf(logger.ErrorLevel, "nacos getconfig failed, err:%v", err)
		return nil, err
	}
	newCs := &source.ChangeSet{
		Data:      []byte(content),
		Format:    "yml",
		Source:    n.String(),
		Timestamp: time.Now(),
	}
	newCs.Checksum = newCs.Sum()
	return newCs, nil
}

// 暂不支持写入配置，只允许通过nacos中心控制
func (n *nacosSource) Write(cs *source.ChangeSet) error {
	return errors.New("nacos config doesn't implement Write method")
}

func (n *nacosSource) Watch() (source.Watcher, error) {
	return newNacosWatcher(n)
}

func (n *nacosSource) String() string {
	return n.param.DataId + " " + n.param.Group
}
