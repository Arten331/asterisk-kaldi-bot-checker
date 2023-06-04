package global

type globals struct {
	serviceName string
	env         string
}

var _g globals //nolint:gochecknoglobals // why not

func AppName() string {
	return _g.serviceName
}

func AppEnv() string {
	return _g.env
}

func SetGlobals(serviceName, env string) {
	_g.serviceName = serviceName
	_g.env = env
}
