package core

import (
	"flag"
	"os"
)

func InitConfig() {
	flag.StringVar(&clusterName, "clustername", "default", "The horae cluster name (env HORAE_CLUSTERNAME takes precedence")
	flag.StringVar(&cassandraAddress, "cassandra-address", "127.0.0.1", "The cassandra address (env HORAE_CASSANDRA_ADDRESS takes precedence)")
	flag.StringVar(&etcdAddress, "etcd-address", "127.0.0.1:4001", "The etcd address/port combination (env HORAE_ETCD_ADDRESS takes precedence)")
	flag.Parse()
	if os.Getenv("HORAE_CLUSTERNAME") != "" {
		clusterName = os.Getenv("HORAE_CLUSTERNAME")
	}
	if os.Getenv("HORAE_CASSANDRA_ADDRESS") != "" {
		cassandraAddress = os.Getenv("HORAE_CASSANDRA_ADDRESS")
	}
	if os.Getenv("HORAE_ETCD_ADDRESS") != "" {
		etcdAddress = os.Getenv("HORAE_ETCD_ADDRESS")
	}
}
