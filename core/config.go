package core

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func InitConfig() {
	flag.Usage = func() {
		fmt.Printf("Usage of horae:\n\nAll params may be applied via OS environment variables as specified below. These\nvariables take precedence over the command line flags. Default port: 8015\n\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(&staticPort, "static-port", true, "Should horae use a static port (HORAE_USE_STATIC_PORT)")
	flag.StringVar(&clusterName, "clustername", "default", "The horae cluster name (HORAE_CLUSTERNAME)")
	flag.StringVar(&cassandraAddress, "cassandra-address", "127.0.0.1", "Our cassandra address (HORAE_CASSANDRA_ADDRESS)")
	flag.StringVar(&etcdAddress, "etcd-address", "127.0.0.1:4001", "Our etcd address/port (HORAE_ETCD_ADDRESS)")
	flag.Parse()
	if os.Getenv("HORAE_USE_STATIC_PORT") != "" {
		value := strings.ToLower(os.Getenv("HORAE_USE_STATIC_PORT"))
		if value == "true" {
			staticPort = true
		} else {
			staticPort = false
		}
	}
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
