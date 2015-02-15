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

func InitDAO(clusterName string) {
	log.WithFields(log.Fields{"cluster": clusterName}).Info("Initializing DB Connection")
	// TODO - need config to define where to find cassandra instance
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "horae_" + clusterName
	sess, err := cluster.CreateSession()
	if err != nil {
		log.WithFields(log.Fields{"reason": err}).Fatal("Unable to init DB connection")
	} else {
		session = sess
	}
}
