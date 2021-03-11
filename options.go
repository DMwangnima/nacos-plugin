package nacos_plugin

import (
	"context"
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

type GetOptions struct {
	vo.GetServiceParam
}

type ClientOption func(*ClientOptions)

type ServerOption func(*ServerOptions)

type GetOption func(*GetOptions)

type ServerNode []ServerOption

type clientKey struct{}

type serverKey struct{}

type getServiceKey struct{}

func LogLevel(level string) ClientOption {
	return func(o *ClientOptions) {
		o.LogLevel = level
	}
}

func NotLoadCacheAtStart(flag bool) ClientOption {
	return func(o *ClientOptions) {
		o.NotLoadCacheAtStart = flag
	}
}

// 该属性是最重要的属性，用于决定命名空间，例如PROD DEV TEST，具体的nsId需要查看nacos
func NamespaceId(nsId string) ClientOption {
	return func(o *ClientOptions) {
		o.NamespaceId = nsId
	}
}

func Endpoint(ep string) ClientOption {
	return func(o *ClientOptions) {
		o.Endpoint = ep
	}
}

func RegionId(rId string) ClientOption {
	return func(o *ClientOptions) {
		o.RegionId = rId
	}
}

func AccessKey(ak string) ClientOption {
	return func(o *ClientOptions) {
		o.AccessKey = ak
	}
}

func SecretKey(sk string) ClientOption {
	return func(o *ClientOptions) {
		o.SecretKey = sk
	}
}

func UserName(un string) ClientOption {
	return func(o *ClientOptions) {
		o.Username = un
	}
}

func Password(pw string) ClientOption {
	return func(o *ClientOptions) {
		o.Password = pw
	}
}

func ContextPath(cp string) ClientOption {
	return func(o *ClientOptions) {
		o.ContextPath = cp
	}
}

func IpAddr(ia string) ServerOption {
	return func(o *ServerOptions) {
		o.IpAddr = ia
	}
}

func Port(p int) ServerOption {
	return func(o *ServerOptions) {
		o.Port = uint64(p)
	}
}

func Clusters(c []string) GetOption {
	return func(o *GetOptions) {
		o.Clusters = c
	}
}

func ServiceName(n string) GetOption {
	return func(o *GetOptions) {
		o.ServiceName = n
	}
}

func GroupName(n string) GetOption {
	return func(o *GetOptions) {
		o.GroupName = n
	}
}

func Client(cliOpts ...ClientOption) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, clientKey{}, cliOpts)
	}
}

func Server(srvNodes ...ServerNode) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, serverKey{}, srvNodes)
	}
}

//func GetServiceParam(getOpts ...GetOption) registry.Option {
//	return func(o *registry.Options) {
//		if o.Context == nil {
//			o.Context = context.Background()
//		}
//		o.Context = context.WithValue(o.Context, getServiceKey{}, getOpts)
//	}
//}
