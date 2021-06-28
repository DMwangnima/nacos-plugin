package registry

import (
	"errors"
	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"strconv"
	"sync"
)

const (
	BUF_NUM = 20
	RETRIES = 3
)

type nacosWatcher struct {
	reg *nacosRegistry
	// 退订重试次数
	retries int
	options *registry.WatchOptions
	mu      sync.RWMutex
	// key为服务名，value为该service的node set
	srvNodeMap map[string]map[string]struct{}

	exit chan bool
	next chan *registry.Result
}

func newNacosWatcher(reg *nacosRegistry, opts ...registry.WatchOption) (registry.Watcher, error) {
	watchOptions := &registry.WatchOptions{}
	for _, opt := range opts {
		opt(watchOptions)
	}
	watcher := &nacosWatcher{
		reg:        reg,
		retries:    RETRIES,
		options:    watchOptions,
		srvNodeMap: make(map[string]map[string]struct{}),
		exit:       make(chan bool),
		next:       make(chan *registry.Result, BUF_NUM),
	}
	go watcher.subscribeServices()
	return watcher, nil
}

func (w *nacosWatcher) subscribeServices() {
	retry := func(service string, function func(*vo.SubscribeParam) error) {
		param := &vo.SubscribeParam{
			ServiceName:       service,
			SubscribeCallback: w.watcherCallback,
		}
		var err error
		var i int
		for i = 0; i < w.retries; i++ {
			err = function(param)
			if err != nil {
				continue
			}
			break
		}
		if i >= w.retries {
			logger.Logf(logger.ErrorLevel, "nacos unsubscribe service %s %d times failed", service, w.retries)
		}
	}

	services := []string{}
	// 退出时取消订阅
	defer func() {
		for _, service := range services {
			retry(service, w.reg.naming.Unsubscribe)
		}
	}()

	for {
		select {
		case service := <-w.reg.serviceChan:
			services = append(services, service)
			w.mu.Lock()
			w.srvNodeMap[service] = make(map[string]struct{})
			w.mu.Unlock()
			go retry(service, w.reg.naming.Subscribe)
		case <-w.exit:
			return
		}
	}
}

func (w *nacosWatcher) watcherCallback(services []model.SubscribeService, err error) {
	if err != nil || len(services) == 0 {
		// 需要使用日志记录
		return
	}
	key := services[0].ServiceName
	oldNodeSet := make(map[string]struct{})
	// 创建一个副本
	w.mu.RLock()
	for k := range w.srvNodeMap[key] {
		oldNodeSet[k] = struct{}{}
	}
	w.mu.RUnlock()

	if len(services) == len(oldNodeSet) {
		var flag bool
		for _, node := range services {
			address := node.Ip + ":" + strconv.Itoa(int(node.Port))
			if _, ok := oldNodeSet[address]; !ok {
				flag = true
				break
			}
		}
		// 表示服务节点无变化
		if !flag {
			return
		}
	}

	newService := &registry.Service{
		Name:  key,
		Nodes: make([]*registry.Node, len(services)),
	}

	for i, node := range services {
		newService.Nodes[i] = &registry.Node{
			Id:       node.InstanceId,
			Address:  node.Ip + ":" + strconv.Itoa(int(node.Port)),
			Metadata: node.Metadata,
		}
	}

	newNodeSet := make(map[string]struct{})
	for _, n := range newService.Nodes {
		newNodeSet[n.Address] = struct{}{}
	}
	w.mu.Lock()
	w.srvNodeMap[key] = newNodeSet
	w.mu.Unlock()

	w.next <- &registry.Result{
		Action:  "update",
		Service: newService,
	}
}

func (w *nacosWatcher) Next() (*registry.Result, error) {
	select {
	case <-w.exit:
		return nil, errors.New("nacosWatcher has been stopped")
	case result, ok := <-w.next:
		if !ok {
			return nil, errors.New("nacos subscribe has something wrong")
		}
		return result, nil
	}
}

func (w *nacosWatcher) Stop() {
	select {
	case <-w.exit:
		return
	default:
		close(w.exit)
	}
}
