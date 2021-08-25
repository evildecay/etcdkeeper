package etcdkeeper

import "flag"

//EtcdkeeperConfig configuration for etcdkeeper object
type EtcdkeeperConfig struct {
	separator      string
	usetls         bool
	cacert         string
	cert           string
	keyfile        string
	useAuth        bool
	connectTimeout int
	grpcMaxMsgSize int
}

// ParseFlag use build in flag module to parse command line etcdkeeper configuration
func (conf *EtcdkeeperConfig) ParseFlag() {
	flag.StringVar(&conf.separator, "sep", "/", "separator")
	flag.BoolVar(&conf.usetls, "usetls", false, "use tls")
	flag.StringVar(&conf.cacert, "cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle (v3)")
	flag.StringVar(&conf.cert, "cert", "", "identify secure client using this TLS certificate file (v3)")
	flag.StringVar(&conf.keyfile, "key", "", "identify secure client using this TLS key file (v3)")
	flag.BoolVar(&conf.useAuth, "auth", false, "use auth")
	flag.IntVar(&conf.connectTimeout, "timeout", 5, "ETCD client connect timeout")
	flag.IntVar(&conf.grpcMaxMsgSize, "grpcMaxMsgSize", 64*1024*1024, "ETCDv3 grpc client max receive msg size")
}
