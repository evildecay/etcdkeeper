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

	// template only
	EtcdkeeperTimeout     int
	EtcdDefaultVersion    int
	EtcdDefaultHostname   string
	EtcdDefaultPort       int
	EtcdV3DefaultTreeMode string
}

// ParseFlag use build in flag module to parse command line etcdkeeper configuration
func (conf *EtcdkeeperConfig) ParseFlag() {
	flag.StringVar(&conf.separator, "sep", "/", "separator")
	flag.BoolVar(&conf.usetls, "usetls", false, "use tls")
	flag.StringVar(&conf.cacert, "cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle (v3)")
	flag.StringVar(&conf.cert, "cert", "", "identify secure client using this TLS certificate file (v3)")
	flag.StringVar(&conf.keyfile, "key", "", "identify secure client using this TLS key file (v3)")
	flag.BoolVar(&conf.useAuth, "auth", false, "use auth")
	flag.IntVar(&conf.connectTimeout, "timeout", 5, "ETCD client connect timeout second")
	flag.IntVar(&conf.grpcMaxMsgSize, "grpc-max-msg-size", 64*1024*1024, "ETCDv3 grpc client max receive msg size")

	flag.IntVar(&conf.EtcdkeeperTimeout, "etcdkeeper-timeout", 5000, "ETCDkeeper frontend connect timeout millisecond")

	// etcd default value
	flag.IntVar(&conf.EtcdDefaultVersion, "etcd-version", 3, "ETCD default version 2 or 3")
	flag.StringVar(&conf.EtcdDefaultHostname, "etcd-hostname", "127.0.0.1", "ETCD default hostname or address")
	flag.IntVar(&conf.EtcdDefaultPort, "etcd-port", 2379, "ETCD default port number")
	flag.StringVar(&conf.EtcdV3DefaultTreeMode, "etcd-default-tree-mode", "list", "ETCD v3 only default display mode, list or path available (v3)")
}
