package config

import (
	"errors"
	"github.com/asim/go-micro/v3/config/source"
	"github.com/asim/go-micro/v3/logger"
	"time"
)

type nacosWatcher struct {
	src      *nacosSource
	confChan chan *source.ChangeSet
	exit     chan struct{}
}

func newNacosWatcher(n *nacosSource) (source.Watcher, error) {
	watcher := &nacosWatcher{
		src:      n,
		confChan: make(chan *source.ChangeSet, 10),
		exit:     make(chan struct{}),
	}
	if err := watcher.subscribe(); err != nil {
		logger.Logf(logger.ErrorLevel, "nacos subscribe config failed, err:%v", err)
		return nil, err
	}
	return watcher, nil
}

func (n *nacosWatcher) subscribe() error {
	n.src.param.OnChange = n.watcherCallback
	return n.src.config.ListenConfig(n.src.param.ConfigParam)
}

func (n *nacosWatcher) watcherCallback(namespace, group, dataId, data string) {
	select {
	case <-n.exit:
		return
	default:
	}
	newCs := &source.ChangeSet{
		Data:      []byte(data),
		Format:    "yml",
		Source:    dataId + " " + group,
		Timestamp: time.Now(),
	}
	newCs.Checksum = newCs.Sum()
	n.confChan <- newCs
}

func (n *nacosWatcher) Next() (*source.ChangeSet, error) {
	select {
	case <-n.exit:
		return nil, errors.New("nacos config watcher has been stopped")
	case cs := <-n.confChan:
		return cs, nil
	}
}

func (n *nacosWatcher) Stop() error {
	select {
	case <-n.exit:
		return errors.New("nacos config watcher has been stopped")
	default:
		close(n.exit)
		return n.src.config.CancelListenConfig(n.src.param.ConfigParam)
	}
}
