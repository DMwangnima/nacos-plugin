package nacos

import (
	"context"
	"errors"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const bufNum = 20

type nacosWatcher struct {
	reg     *nacosRegistry
	options *registry.WatchOptions

	exit chan bool
	next chan *registry.Result

	services []*registry.Service
	mu       sync.RWMutex
	cache    map[string][]model.SubscribeService
}

func newNacosWatcher(reg *nacosRegistry, opts ...registry.WatchOption) (registry.Watcher, error) {
	watchOptions := &registry.WatchOptions{}
	for _, opt := range opts {
		opt(watchOptions)
	}
	watcher := &nacosWatcher{
		reg:     reg,
		options: watchOptions,
		exit:    make(chan bool),
		next:    make(chan *registry.Result, bufNum),
		cache:   make(map[string][]model.SubscribeService),
	}
	if err := subscribeServices(watcher); err != nil {
		return nil, err
	}
	return watcher, nil
}

func subscribeServices(watcher *nacosWatcher) error {
	services, err := watcher.reg.ListServices()
	if err != nil {
		return errors.New("invoke nacosRegistry.ListServices failed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finish := make(chan struct{}, len(services))
	notFinish := make(chan struct{}, len(services))
	// 广播成功信号
	success := make(chan struct{})
	subscribe := func(ctx context.Context, serv *registry.Service) {
		param := &vo.SubscribeParam{
			ServiceName:       serv.Name,
			Clusters:          nil,
			GroupName:         "",
			SubscribeCallback: watcher.watcherCallback,
		}
		if err = watcher.reg.naming.Subscribe(param); err == nil {
			finish <- struct{}{}
			select {
			case <-success:
				return
			case <-ctx.Done():
				_ = watcher.reg.naming.Unsubscribe(param)
				// 考虑指数回退重试
			}
		} else {
			notFinish <- struct{}{}
			<-ctx.Done()
		}
	}

	for _, service := range services {
		go subscribe(ctx, service)
	}

	timer := time.NewTimer(10 * time.Second)
	var cnt int32
	for {
		select {
		case <-notFinish:
			return errors.New("subscribe failed")
		case <-finish:
			if newVal := atomic.AddInt32(&cnt, 1); int(newVal) == len(services) {
				close(success)
				watcher.services = services
				return nil
			}
		case <-timer.C:
			return errors.New("subscribe timeout")
		}
	}
}

func (w *nacosWatcher) watcherCallback(services []model.SubscribeService, err error) {
	if err != nil || len(services) == 0 {
		// 需要使用日志记录
		return
	}
	key := services[0].ServiceName
	w.mu.Lock()
	var action string
	if _, ok := w.cache[key]; !ok {
		action = "create"
	} else {
		action = "update"
	}
	w.cache[key] = services
	w.mu.Unlock()
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
	w.next <- &registry.Result{
		Action:  action,
		Service: newService,
	}
}

func (w *nacosWatcher) Next() (*registry.Result, error) {
	select {
	case <-w.exit:
		return nil, errors.New("nacosWatcher has been stopped")
	case result, ok := <-w.next:
		if !ok {
			return nil, errors.New("nacosWatcher has been stopped")
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
		close(w.next)
		for _, service := range w.services {
			param := &vo.SubscribeParam{
				ServiceName:       service.Name,
				Clusters:          nil,
				GroupName:         "",
				SubscribeCallback: w.watcherCallback,
			}
			_ = w.reg.naming.Unsubscribe(param)
		}
	}
}
