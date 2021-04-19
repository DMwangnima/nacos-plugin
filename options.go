package nacos

import (
	"context"
	"github.com/asim/go-micro/v3/config/source"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type ClientOptions struct {
	constant.ClientConfig
}

type ServerOptions struct {
	constant.ServerConfig
}

type InstanceOptions struct {
	vo.RegisterInstanceParam
}

type ConfigOptions struct {
	vo.ConfigParam
}

type ClientOption func(*ClientOptions)

type ServerOption func(*ServerOptions)

type InstanceOption func(*InstanceOptions)

type ConfigOption func(*ConfigOptions)

type ServerNode []ServerOption

type ClientKey struct{}

type ServerKey struct{}

type InstanceKey struct{}

type ConfParamKey struct{}

// Client配置项
func TimeoutMs(time uint64) ClientOption {
	return func(o *ClientOptions) {
		o.TimeoutMs = time
	}
}

func BeatInterval(time int64) ClientOption {
	return func(o *ClientOptions) {
		o.BeatInterval = time
	}
}

// 该属性是最重要的属性，用于决定命名空间，例如PROD DEV TEST，具体的nsId需要查看nacos
func NamespaceId(id string) ClientOption {
	return func(o *ClientOptions) {
		o.NamespaceId = id
	}
}

func Endpoint(point string) ClientOption {
	return func(o *ClientOptions) {
		o.Endpoint = point
	}
}

func RegionId(id string) ClientOption {
	return func(o *ClientOptions) {
		o.RegionId = id
	}
}

func AccessKey(key string) ClientOption {
	return func(o *ClientOptions) {
		o.AccessKey = key
	}
}

func SecretKey(key string) ClientOption {
	return func(o *ClientOptions) {
		o.SecretKey = key
	}
}

func OpenKMS(flag bool) ClientOption {
	return func(o *ClientOptions) {
		o.OpenKMS = flag
	}
}

func CacheDir(dir string) ClientOption {
	return func(o *ClientOptions) {
		o.CacheDir = dir
	}
}

func UpdateThreadNum(num int) ClientOption {
	return func(o *ClientOptions) {
		o.UpdateThreadNum = num
	}
}

func NotLoadCacheAtStart(flag bool) ClientOption {
	return func(o *ClientOptions) {
		o.NotLoadCacheAtStart = flag
	}
}

func UpdateCacheWhenEmpty(flag bool) ClientOption {
	return func(o *ClientOptions) {
		o.UpdateCacheWhenEmpty = flag
	}
}

func UserName(name string) ClientOption {
	return func(o *ClientOptions) {
		o.Username = name
	}
}

func Password(pw string) ClientOption {
	return func(o *ClientOptions) {
		o.Password = pw
	}
}

func LogDir(dir string) ClientOption {
	return func(o *ClientOptions) {
		o.LogDir = dir
	}
}

func RotateTime(time string) ClientOption {
	return func(o *ClientOptions) {
		o.RotateTime = time
	}
}

func MaxAge(val int64) ClientOption {
	return func(o *ClientOptions) {
		o.MaxAge = val
	}
}

func LogLevel(level string) ClientOption {
	return func(o *ClientOptions) {
		o.LogLevel = level
	}
}

func ClientContextPath(path string) ClientOption {
	return func(o *ClientOptions) {
		o.ContextPath = path
	}
}

// Server配置项
func Scheme(s string) ServerOption {
	return func(o *ServerOptions) {
		o.Scheme = s
	}
}

func ServerContextPath(path string) ServerOption {
	return func(o *ServerOptions) {
		o.ContextPath = path
	}
}

func IpAddr(ip string) ServerOption {
	return func(o *ServerOptions) {
		o.IpAddr = ip
	}
}

func Port(p int) ServerOption {
	return func(o *ServerOptions) {
		o.Port = uint64(p)
	}
}

func Weight(w float64) InstanceOption {
	return func(o *InstanceOptions) {
		o.Weight = w
	}
}

func Enable(flag bool) InstanceOption {
	return func(o *InstanceOptions) {
		o.Enable = flag
	}
}

func Healthy(flag bool) InstanceOption {
	return func(o *InstanceOptions) {
		o.Healthy = flag
	}
}

func MetaData(md map[string]string) InstanceOption {
	newMd := make(map[string]string)
	for k, v := range md {
		newMd[k] = v
	}
	return func(o *InstanceOptions) {
		o.Metadata = newMd
	}
}

func ClusterName(n string) InstanceOption {
	return func(o *InstanceOptions) {
		o.ClusterName = n
	}
}

func ServiceName(n string) InstanceOption {
	return func(o *InstanceOptions) {
		o.ServiceName = n
	}
}

func GroupName(n string) InstanceOption {
	return func(o *InstanceOptions) {
		o.GroupName = n
	}
}

func Ephemeral(flag bool) InstanceOption {
	return func(o *InstanceOptions) {
		o.Ephemeral = flag
	}
}

func DataId(id string) ConfigOption {
	return func(o *ConfigOptions) {
		o.DataId = id
	}
}

func Group(g string) ConfigOption {
	return func(o *ConfigOptions) {
		o.Group = g
	}
}

func Client(cliOpts ...ClientOption) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, ClientKey{}, cliOpts)
	}
}

func Server(srvNodes ...ServerNode) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, ServerKey{}, srvNodes)
	}
}

func Instance(insOpts ...InstanceOption) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, InstanceKey{}, insOpts)
	}
}

func ConfClient(cliOpts ...ClientOption) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, ClientKey{}, cliOpts)
	}
}

func ConfServer(srvNodes ...ServerNode) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, ServerKey{}, srvNodes)
	}
}

func ConfParam(confOpts ...ConfigOption) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, ConfParamKey{}, confOpts)
	}
}
