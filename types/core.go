package types

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
)

var session *gocql.Session

// type defines the core data set of the running node
type Node struct {
	UUID    gocql.UUID
	Cluster string
	Address string
	Port    string
}

func InitDAO(cassandraAddress string, clusterName string) {
	log.WithFields(log.Fields{"cluster": clusterName}).Info("Initializing DB Connection")
	cluster := gocql.NewCluster(cassandraAddress)
	cluster.Keyspace = "horae_" + clusterName
	sess, err := cluster.CreateSession()
	if err != nil {
		log.WithFields(log.Fields{"reason": err}).Fatal("Unable to init DB connection")
	} else {
		session = sess
	}
}
