package types

import (
	"github.com/gocql/gocql"
)

const (
	EunomiaQueue                     = "queue"
	EunomiaTask                      = "task"
	EunomiaActionUpdate              = "action_update"
	EunomiaActionDelete              = "action_delete"
	EunomiaActionMonitor             = "action_monitor"
	EunomiaRequestBecomeMaster       = "state_master"
	EunomiaRequestReleaseMaster      = "state_release"
	EunomiaResponseBecameQueueMaster = "became_queue_master"
	EunomiaResponseBecameQueueSlave  = "became_queue_slave"
	EunomiaResponseActionCreate      = "response_action_create"
	EunomiaResponseActionUpdate      = "response_action_update"
	EunomiaResponseActionDelete      = "response_action_delete"
)

type EunomiaRequest struct {
	Action                  string // update / delete / monitor
	Key                     string // key/value updates via workers in Eunomia
	Value                   string
	TTL                     uint64
	QueueUUID               gocql.UUID // additional fields required for queue monitor setup
	ChannelFromQueueManager chan EunomiaQueueRequest
	ChannelToQueueManager   chan EunomiaQueueResponse
}

type EunomiaQueueRequest struct {
	Action    string
	QueueUUID gocql.UUID
}

type EunomiaQueueResponse struct {
	Type   string     // queue/task
	Action string     // create/update/delete
	UUID   gocql.UUID // UUID of changing object
}
