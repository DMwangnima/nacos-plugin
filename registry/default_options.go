package registry

type options struct {
	serviceName string
	ephemeral   bool
	metaData    map[string]string
}

type Option func(*options)

func WithServiceName(n string) Option {
	return func(o *options) {
		o.serviceName = n
	}
}

func WithEphemeral(e bool) Option {
	return func(o *options) {
		o.ephemeral = e
	}
}

func WithMetaData(m map[string]string) Option {
	newMd := make(map[string]string)
	for k, v := range m {
		newMd[k] = v
	}
	return func(o *options) {
		o.metaData = newMd
	}
}