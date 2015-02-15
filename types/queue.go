package types

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
)

const (
	QueueSync  = "sync"
	QueueAsync = "async"
)

type Queue struct {
	UUID                   gocql.UUID `cql:"queue_uuid" json:"uuid,required" description:"The unique identifier of the queue"`
	Name                   string     `cql:"name" json:"name,omitempty" description:"The unique name of the queue"`
	QueueType              string     `cql:"queue_type" json:"queueType,omitempty" description:"The type of queue: sync or async"`
	WindowOfOperation      string     `cql:"window_of_operation" json:"windowOfOperation,omitempty" description:"The window of operation for the queue if defined as sync"`
	ShouldDrain            bool       `cql:"should_drain" json:"shouldDrain,omitempty" description:"The expected behaviour of the queue when it is deleted. If true the queue will drain (and no longer accept new requests) before it is deleted.  Defaults to true"`
	BackPressureAction     string     `cql:"backpressure_action" json:"backpressureAction,omitempty" description:"The unique identifier of an action to be called in the event that the backpressure definition is breached"`
	BackpressureDefinition string     `cql:"backpressure_definition" json:"backpressureDefinition,omitempty" description:"For synchronous queues the backpressure definition defines the number of waiting task slots before the backpressure API endpoint is called."`
	Tasks                  []Task     `json:"-"`
}

func GetQueues() []Queue {
	query := session.Query("select * from queues")
	bind := cqlr.BindQuery(query)
	var queue Queue
	queues := []Queue{}
	for bind.Scan(&queue) {
		queues = append(queues, queue)
	}
	return queues
}

func GetQueuesByTag(tag string) []Queue {
	var id gocql.UUID
	var queue Queue
	queues := []Queue{}
	iteration := session.Query("select object_uuid from tags where type = 'queue' and tag = ? allow filtering", tag).Iter()
	for iteration.Scan(&id) {
		q := session.Query("select * from queues where queue_uuid = ?", id)
		b := cqlr.BindQuery(q)
		b.Scan(&queue)
		queues = append(queues, queue)
	}
	return queues
}

func GetQueue(queueUUID string) Queue {
	query := session.Query("select * from queues where queue_uuid = ?", queueUUID)
	bind := cqlr.BindQuery(query)
	var queue Queue
	if !bind.Scan(&queue) {
		log.Println("didnt match anything....")
	}
	return queue
}

func GetQueueByPath(path string) (Queue, error) {
	var id gocql.UUID
	if err := session.Query(`select queue_uuid from paths where path = ? limit 1 allow filtering`, path).Scan(&id); err != nil {
		return Queue{}, errors.New("No queue found")
	}
	query := session.Query("SELECT * FROM queues where queue_uuid = ?", id)
	bind := cqlr.BindQuery(query)
	var queue Queue
	bind.Scan(&queue)
	return queue, nil
}

func (queue Queue) CreateOrUpdate() error {
	// ensure paths are unique for object
	bind := cqlr.Bind(`insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain) values (?, ?, ?, ?, ?)`, queue)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (queue Queue) DeleteQueue() error {
	bind := cqlr.Bind(`delete from queues where queue_uuid = ?`, queue)
	if err := bind.Exec(session); err != nil {
		log.Print("received error from delete")
		return err
	} else {
		return nil
	}
}

func (q Queue) Tags() []string {
	return GetTagsForObject(q.UUID)
}

func (q Queue) SetTags(tags []string) {
	SetTagsForObject(q.UUID, tags, "queue")
}

func (q Queue) Paths() []string {
	// find and return paths for queue
	return GetPathsForQueue(q)
}

func (q Queue) SetPaths(paths []string) {
	// set paths on queue
	SetPathsForQueue(q, paths)
}

func GetPathsForQueue(queue Queue) []string {
	paths := []string{}
	path := ""
	iteration := session.Query("select path from paths where queue_uuid = ?", queue.UUID).Iter()
	for iteration.Scan(&path) {
		paths = append(paths, path)
	}
	return paths
}

func SetPathsForQueue(queue Queue, paths []string) error {
	for _, path := range paths {
		if err := session.Query(`insert into paths (queue_uuid, path) VALUES (?, ?)`, queue.UUID, path).Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (q *Queue) LoadTasks() {
	log.WithFields(log.Fields{"name": q.Name}).Info("Loading tasks on Queue")
	// TODO - should order tasks by execution time and priority
	if q.Name == "root" {
		// root never receives tasks
		q.Tasks = []Task{}
	} else {
		uuid := gocql.UUID{}
		iteration := session.Query("select task_uuid from tasks where queue_uuid = ?", q.UUID).Iter()
		for iteration.Scan(&uuid) {
			q.Tasks = append(q.Tasks, GetTaskWithQueue(q.UUID, uuid))
		}
	}
}

// TODO - fix all of this below

func (q Queue) Contained() bool {
	// return true or false if queue is currently contained
	return true
}

func (q *Queue) StartExecution() {
	log.WithFields(log.Fields{"name": q.Name}).Info("Starting execution on Queue")

	//if q.QueueType == QueueSync {
	// execute task at top of the list
	// wait for completion - seen via etcd update
	// then start the next

	//} else {
	// async queue.  find first task
	for _, task := range q.Tasks {
		task.Execute(false)
	}

	// if queue is async

	// every minute do the following: for all tasks executing in the next minute start timers to execute
}

func (q *Queue) StopExecution() {
	log.WithFields(log.Fields{"name": q.Name}).Info("Stopping execution on Queue")
}
