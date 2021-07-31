package config

import (
	"github.com/asim/go-micro/v3/config"
	"testing"
)

type Gateway struct {
	Redis      Redis      `json:"redis"`
	FileServer FileServer `json:"fileServer"`
	Kafka      Kafka      `json:"kafka"`
}

type Redis struct {
	Addrs []string `json:"addrs"`
}

type Kafka struct {
	Addrs []string `json:"addrs"`
}

type FileServer struct {
	Addr string `json:"addr"`
}

func TestNewDefaultSource(t *testing.T) {
	sour := NewDefaultSource("gateway")
	conf, _ := config.NewConfig()
	if err := conf.Load(sour); err != nil {
		t.Log(err)
		return
	}
	g := &Gateway{}
	err := conf.Scan(g)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(g)
}
