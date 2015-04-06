package types

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var Configuration Config

type Config struct {
	ClusterName      string
	CassandraAddress string
	ETCDAddress      string
	StaticPort       bool
	MasterURI        string
}

func InitConfig() {
	Configuration = Config{}

	flag.Usage = func() {
		fmt.Printf("Usage of horae:\n\nAll params may be applied via OS environment variables as specified below. These\nvariables take precedence over the command line flags. Default port: 8015\n\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(&Configuration.StaticPort, "static-port", true, "Should horae use a static port (HORAE_USE_STATIC_PORT)")
	flag.StringVar(&Configuration.ClusterName, "clustername", "default", "The horae cluster name (HORAE_CLUSTERNAME)")
	flag.StringVar(&Configuration.CassandraAddress, "cassandra-address", "127.0.0.1", "Our cassandra address (HORAE_CASSANDRA_ADDRESS)")
	flag.StringVar(&Configuration.ETCDAddress, "etcd-address", "127.0.0.1:4001", "Our etcd address/port (HORAE_ETCD_ADDRESS)")
	flag.Parse()
	if os.Getenv("HORAE_USE_STATIC_PORT") != "" {
		value := strings.ToLower(os.Getenv("HORAE_USE_STATIC_PORT"))
		if value == "true" {
			Configuration.StaticPort = true
		} else {
			Configuration.StaticPort = false
		}
	}
	if os.Getenv("HORAE_CLUSTERNAME") != "" {
		Configuration.ClusterName = os.Getenv("HORAE_CLUSTERNAME")
	}
	if os.Getenv("HORAE_CASSANDRA_ADDRESS") != "" {
		Configuration.CassandraAddress = os.Getenv("HORAE_CASSANDRA_ADDRESS")
	}
	if os.Getenv("HORAE_ETCD_ADDRESS") != "" {
		Configuration.ETCDAddress = os.Getenv("HORAE_ETCD_ADDRESS")
	}
}
