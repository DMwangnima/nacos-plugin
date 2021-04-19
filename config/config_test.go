package config

import (
	"fmt"
	"github.com/DMwangnima/nacos-plugin"
	"github.com/asim/go-micro/v3/config"
	"github.com/asim/go-micro/v3/config/source"
	"testing"
)

func newSource() source.Source {
	cli := nacos.ConfClient(
		nacos.NamespaceId("a9c64542-87bf-432e-816f-4bd5c19806ac"),
	)
	node := nacos.ServerNode{
		nacos.IpAddr("192.168.3.7"),
		nacos.Port(8848),
	}
	node1 := nacos.ServerNode{
		nacos.IpAddr("192.168.3.15"),
		nacos.Port(8848),
	}
	node2 := nacos.ServerNode{
		nacos.IpAddr("192.168.3.16"),
		nacos.Port(8848),
	}
	srv := nacos.ConfServer(node, node1, node2)
	private := nacos.ConfParam(
		nacos.Group("DEFAULT_GROUP"),
		nacos.DataId("hello-test"),
	)
	conf := NewSource(cli, srv, private)
	return conf
}

func TestRead(t *testing.T) {
	conf := newSource()
	content, err := conf.Read()
	fmt.Println(content)
	fmt.Println(err)
}

func TestWatch(t *testing.T) {
	conf := newSource()
	watcher, err := conf.Watch()
	if err != nil {
		fmt.Println(err)
		return
	}
	cs, err := watcher.Next()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(cs)
	err = watcher.Stop()
	if err != nil {
		fmt.Println(err)
	}
	cs, err = watcher.Next()
	fmt.Println(err)
}

type ConfigType struct {
	Redis1 redisConf `json:"redis1,omitempty"`
	Redis2 redisConf `json:"redis2,omitempty"`
}

type redisConf struct {
	Ip   string `json:"ip,omitempty"`
	Port int    `json:"port,omitempty"`
}

func TestConfig(t *testing.T) {
	sour := newSource()
	conf, _ := config.NewConfig()
	if err := conf.Load(sour); err != nil {
		fmt.Println(err)
		return
	}
	c := &ConfigType{}
	err := conf.Scan(c)
	if err != nil {
		fmt.Println(err)
		return
	}
}
