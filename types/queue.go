package types

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
	"time"
)

const (
	QueueSync     = "sync"
	QueueAsync    = "async"
	QueueActive   = "Active"
	QueueDeleted  = "Deleted"
	QueueDeleting = "Deleting"
)

type Queue struct {
	UUID                   gocql.UUID             `cql:"queue_uuid" json:"uuid,required" description:"The unique identifier of the queue"`
	Name                   string                 `cql:"name" json:"name,omitempty" description:"The unique name of the queue"`
	QueueType              string                 `cql:"queue_type" json:"queueType,omitempty" description:"The type of queue: sync or async"`
	WindowOfOperation      string                 `cql:"window_of_operation" json:"windowOfOperation,omitempty" description:"The window of operation for the queue if defined as sync"`
	ShouldDrain            bool                   `cql:"should_drain" json:"shouldDrain,omitempty" description:"The expected behaviour of the queue when it is deleted. If true the queue will drain (and no longer accept new requests) before it is deleted.  Defaults to true"`
	BackPressureAction     *gocql.UUID            `cql:"backpressure_action" json:"backpressureAction,omitempty" description:"The unique identifier of an action to be called in the event that the backpressure definition is breached"`
	BackpressureDefinition uint64                 `cql:"backpressure_definition" json:"backpressureDefinition,omitempty" description:"For queues the backpressure definition defines the number of waiting task slots before the backpressure API endpoint is called."`
	OurTags                []string               `json:"tags,omitempty" description:"Tags assigned to the queue."`
	OurPaths               []string               `json:"paths,omitempty" description:"Paths assigned to the queue."`
	Tasks                  []Task                 `json:"-"`
	Window                 Window                 `json:"-"`
	Running                bool                   `json:"-"`
	Status                 string                 `json:"-"`
	asyncTimerMap          map[string]*time.Timer `json:"-"`
	asyncTimeWindow        time.Time              `json:"-"`
}

// Query
func GetQueues() []Queue {
	query := session.Query("select * from queues where status in (?, ?) allow filtering", QueueActive, QueueDeleting)
	bind := cqlr.BindQuery(query)
	var queue Queue
	queues := []Queue{}
	for bind.Scan(&queue) {
		queue.LoadTags()
		queue.LoadPaths()
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
		queue.LoadTags()
		queue.LoadPaths()
		queues = append(queues, queue)
	}
	return queues
}

func GetQueue(queueUUID string) (Queue, error) {
	query := session.Query("select * from queues where queue_uuid = ?", queueUUID)
	bind := cqlr.BindQuery(query)
	var queue Queue
	if !bind.Scan(&queue) {
		return Queue{}, errors.New("Unknown queue")
	}
	queue.LoadTags()
	queue.LoadPaths()
	return queue, nil
}

func GetQueueByPath(path string) (Queue, error) {
	var id gocql.UUID
	if err := session.Query(`select queue_uuid from paths where path = ? limit 1 allow filtering`, path).Scan(&id); err != nil {
		return Queue{}, errors.New("No queue found")
	}
	query := session.Query("select * from queues where queue_uuid = ?", id)
	bind := cqlr.BindQuery(query)
	var queue Queue
	bind.Scan(&queue)
	queue.LoadTags()
	queue.LoadPaths()
	return queue, nil
}

// CRUD
func (queue *Queue) CreateOrUpdate() error {
	// ensure paths are unique for object
	if queue.UUID.String() == "00000000-0000-0000-0000-000000000000" {
		// queue was generated from json with an unknown UUID.  Fix up
		queue.UUID = gocql.TimeUUID()
	}
	if queue.QueueType != "sync" && queue.QueueType != "async" {
		return errors.New("Invalid queue type")
	}
	_, parseErr := Parse(queue.WindowOfOperation)
	if parseErr != nil {
		return errors.New("Invalid window definition: " + parseErr.Error())
	}
	pathErr := queue.CreateOrUpdatePaths()
	if pathErr != nil {
		return pathErr
	}
	queue.CreateOrUpdateTags()
	queue.Status = QueueActive
	bind := cqlr.Bind(`insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, backpressure_action, backpressure_definition, status) values (?, ?, ?, ?, ?, ?, ?, ?)`, queue)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (queue Queue) Delete() error {
	queue.DeletePaths()
	queue.DeleteTags()
	if err := session.Query(`delete from queues where queue_uuid = ? and status = ?`, queue.UUID, QueueActive).Exec(); err != nil {
		return err
	}
	// add the queue back into the DB with a Deleted/Deleting status.
	if queue.ShouldDrain == true {
		queue.Status = QueueDeleting
	} else {
		queue.Status = QueueDeleted
	}
	bind := cqlr.Bind(`insert into queues (queue_uuid, name, queue_type, window_of_operation, should_drain, backpressure_action, backpressure_definition, status) values (?, ?, ?, ?, ?, ?, ?, ?)`, queue)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (q *Queue) LoadWindow() error {
	window, parseErr := Parse(q.WindowOfOperation)
	if parseErr != nil {
		// Realistically we should never reach this point. CreateOrUpdate will ensure it is valid
		return parseErr
	}
	q.Window = window
	return nil
}

func (q *Queue) LoadTags() {
	q.OurTags = GetTagsForObject(q.UUID)
}

func (q Queue) CreateOrUpdateTags() {
	SetTagsForObject(q.UUID, q.OurTags, "queue")
}

func (q Queue) DeleteTags() {
	DeleteTagsForObject(q.UUID)
}

func (q *Queue) LoadPaths() {
	// find and return paths for queue
	q.OurPaths = LoadPathsFromDB(q.UUID)
}

func LoadPathsFromDB(uuid gocql.UUID) []string {
	// find and return paths for queue
	paths := []string{}
	path := ""
	iteration := session.Query("select path from paths where queue_uuid = ?", uuid).Iter()
	for iteration.Scan(&path) {
		paths = append(paths, path)
	}
	return paths
}

func (q Queue) CreateOrUpdatePaths() error {
	// set paths on queue
	pathsFromDB := LoadPathsFromDB(q.UUID)
	for _, path := range q.OurPaths {
		if path == "/" {
			return errors.New("Cannot define queue with root path")
		}
		if path[len(path)-1:] == "/" {
			return errors.New("Cannot define path with trailing slash")
		}
		if isStringInSlice(path, pathsFromDB) {
			pathsFromDB = findAndRemoveInSlice(path, pathsFromDB)
		} else {
			session.Query(`insert into paths (queue_uuid, path) VALUES (?, ?)`, q.UUID, path).Exec()
		}
	}
	for _, pathToDelete := range pathsFromDB {
		session.Query(`delete from paths where queue_uuid = ? and path = ?`, q.UUID, pathToDelete).Exec()
	}
	return nil
}

func (q Queue) DeletePaths() {
	// delete paths on queue
	session.Query(`delete from paths where queue_uuid = ?`, q.UUID).Exec()
}

// Queue Execution
func (q *Queue) IsRunning() bool {
	return q.Running
}

func (q Queue) MatchesPath(path string) bool {
	for _, p := range q.OurPaths {
		if p == path {
			return true
		}
	}
	return false
}

func (q Queue) CountOfTasks() uint64 {
	var count uint64
	query := "select count(*) from async_tasks where queue_uuid = ? and status = 'Pending' limit 1000000"
	if q.QueueType == QueueSync {
		query = "select count(*) from sync_tasks where queue_uuid = ? and status = 'Pending' limit 1000000"
	}
	if err := session.Query(query, q.UUID).Scan(&count); err == nil {
		return count
	}
	return 0
}

func (q Queue) CheckBackpressure() {
	if q.CountOfTasks() > q.BackpressureDefinition {
		// the depth of the queue exceeds our expectations.
		if q.BackPressureAction.String() == "00000000-0000-0000-0000-000000000000" {
			task, err := GetTask(q.BackPressureAction.String())
			if err == nil {
				task.Execute(true)
			}
		}
	}
}

func (q Queue) ReceivedCompletionForTask(task_uuid string) {
	// in a sync model we only execute a promise (if defined) when a completion message is received.
	task, err := GetTask(task_uuid)
	if err == nil {
		task.ExecutePromise()
	}
}

func (q *Queue) UpdatedTask(action string, task_uuid string) {
	// received task update from queue manager, ignore if we are a sync queue; we're only ever executing one
	// task at a time, and if this change is for an active task it's simply too late to make changes
	if q.QueueType == QueueAsync {
		if action == EunomiaActionCreate {
			t, err := GetTask(task_uuid)
			if err == nil {
				if !q.asyncTimeWindow.IsZero() && t.When.Before(q.asyncTimeWindow) {
					// task is within scope of the current time slice
					q.addToTimerMap(task_uuid)
				}
			}
		} else if action == EunomiaActionUpdate {
			// re-read from DB and set up the timer. there is the potential for an issue here:
			// the task may be executing but the key isn't remove from the map until this has
			// completed.  the chances of this are slim but greater than 0
			q.removeFromTimerMap(task_uuid)
			q.addToTimerMap(task_uuid)
		} else if action == EunomiaActionDelete {
			// if currently in our asyncTimerMap, stop timer and remove
			_, ok := q.asyncTimerMap[task_uuid]
			if ok {
				// this task is known to us
				q.removeFromTimerMap(task_uuid)
			}
		}
	}
}

func (q *Queue) addToTimerMap(task string) {
	t, err := GetTask(task)
	if err == nil {
		// execute task at specified time.
		q.asyncTimerMap[t.UUID.String()] = time.AfterFunc(t.When.Sub(time.Now()), func() {
			t.Execute(false)
			delete(q.asyncTimerMap, t.UUID.String())
		})
	}
}

func (q *Queue) removeFromTimerMap(task string) {
	timer := q.asyncTimerMap[task]
	timer.Stop()
	delete(q.asyncTimerMap, task)
}

func (q *Queue) StartOrContinueExecution(starting bool) {
	if starting {
		log.WithFields(log.Fields{"name": q.Name, "UUID": q.UUID.String()}).Info("Starting execution on Queue")
	} else {
		log.WithFields(log.Fields{"name": q.Name, "UUID": q.UUID.String()}).Info("Continuing execution on Queue")
	}
	q.Running = true
	if q.QueueType == QueueSync {
		// sync mode
		// execute each task in order.  wait for completion and then execute the next
		for {
			var id gocql.UUID
			if err := session.Query(`select task_uuid from sync_tasks where queue_uuid = ? and status = ? limit 1;`, q.UUID, TaskPending).Scan(&id); err == nil {
				// found a valid task in the queue.  execute and return.  we'll rely on the queue manager to start us up again when the completion message is received
				task, err := GetTask(id.String())
				if err == nil {
					// found a matching task. now execute and immediately return
					if !task.Execute(true) {
						// the action failed.  which means we wont ever receive a completion message
						task.ExecutePromise()
						continue
					}
					return
				}
			}
			// we didnt find a task for this queue in scope.  so wait, and try again.
			time.Sleep(15 * time.Second)
		}
	} else if q.QueueType == QueueAsync {
		// async mode
		// execute each task independently based on timestamp
		// reset the timer map
		q.asyncTimerMap = make(map[string]*time.Timer)
		for {
			timeForQuery := time.Now().Add(5 * time.Minute)
			if timeForQuery.After(q.Window.GetNextEndTime()) {
				timeForQuery = q.Window.GetNextEndTime()
			}
			q.asyncTimeWindow = timeForQuery
			var id gocql.UUID
			iteration := session.Query("select task_uuid from async_tasks where queue_uuid = ? and status = ? and when > ? and when < ?", q.UUID, TaskPending, time.Now(), timeForQuery).Iter()
			for iteration.Scan(&id) {
				_, ok := q.asyncTimerMap[id.String()]
				if !ok {
					// we don't currently know about this task.
					q.addToTimerMap(id.String())
				}
			}
			// every 4 minutes, check for new tasks
			time.Sleep(4 * time.Minute)
		}
	}
}

func (q *Queue) StopExecution(reason string) {
	log.WithFields(log.Fields{"name": q.Name, "reason": reason}).Info("Stopping execution on Queue")
	q.Running = false
	if q.QueueType == QueueAsync {
		// stop all the existing tasks in flight
		for k := range q.asyncTimerMap {
			q.removeFromTimerMap(k)
		}
	}
}
