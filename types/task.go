package types

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
	"time"
)

const (
	TaskGet             = "GET"
	TaskPut             = "PUT"
	TaskPost            = "POST"
	TaskHead            = "HEAD"
	TaskDelete          = "DELETE"
	TaskPending         = "Pending"
	TaskRunning         = "Running"
	TaskComplete        = "Complete"
	TaskFailed          = "Failure"
	TaskPartiallyFailed = "Partially Failed"
	TaskDeleted         = "Deleted"
)

type Task struct {
	UUID            gocql.UUID  `cql:"task_uuid" json:"uuid,required" description:"The unique identifier of the task"`
	Name            string      `cql:"name" json:"name,omitempty" description:"The name of the task"`
	Priority        uint64      `cql:"priority" json:"priority,omitempty" description:"The priority of the task. If the queue is sync ordered by priority otherwise ordered by exec time and then priority"`
	Queue           *gocql.UUID `cql:"queue_uuid" json:"queue,omitempty" description:"The UUID of the hosting queue"`
	When            time.Time   `cql:"when" json:"when,omitempty" description:"The future execution timestamp of the task"`
	PromiseAction   *gocql.UUID `cql:"promise_action" json:"promise,omitempty" description:"The unique identifier of the promise, executed on successful completion of the execution action"`
	ExecutionAction *gocql.UUID `cql:"execution_action" json:"execution,required" description:"The unique identifier of the executing action"`
	Status          string      `cql:"status" json:"status,required" description:"The status of the task (Pending/Running/Complete/Failed/Partially Failed)"`
	OurTags         []string    `json:"tags,omitempty" description:"Tags assigned to the task."`
	Promise         Action      `json:"-"`
	Execution       Action      `json:"-"`
}

func GetTasks() []Task {
	query := session.Query("select * from tasks")
	bind := cqlr.BindQuery(query)
	var task Task
	tasks := []Task{}
	for bind.Scan(&task) {
		task.LoadTags()
		tasks = append(tasks, task)
	}
	return tasks
}

func GetTasksByTag(tag string) []Task {
	var id gocql.UUID
	var task Task
	tasks := []Task{}
	iteration := session.Query("select object_uuid from tags where type = 'task' and tag = ? allow filtering", tag).Iter()
	for iteration.Scan(&id) {
		q := session.Query("select * from tasks where task_uuid = ?", id)
		b := cqlr.BindQuery(q)
		b.Scan(&task)
		task.LoadTags()
		tasks = append(tasks, task)
	}
	return tasks
}

func GetTasksByQueue(queue string) []Task {
	var id gocql.UUID
	var task Task
	tasks := []Task{}
	q, err := GetQueue(queue)
	if err == nil {
		table := "async_tasks"
		if q.QueueType == QueueSync {
			table = "sync_tasks"
		}
		iteration := session.Query("select task_uuid from ? where queue_uuid = ? and status in ('Pending', 'Running', 'Complete', 'Failure', 'Partially Failed', 'Deleted')", table, queue).Iter()
		for iteration.Scan(&id) {
			q := session.Query("select * from tasks where task_uuid = ?", id)
			b := cqlr.BindQuery(q)
			b.Scan(&task)
			task.LoadTags()
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func GetTask(taskUUID string) (Task, error) {
	query := session.Query("select * from tasks where task_uuid = ?", taskUUID)
	return bindActionsToTask(query)
}

func bindActionsToTask(query *gocql.Query) (Task, error) {
	bind := cqlr.BindQuery(query)
	var task Task
	if !bind.Scan(&task) {
		return Task{}, errors.New("Unknown task")
	}
	task.LoadTags()
	// at this point we have the Task object. Now find associated actions
	if task.PromiseAction != nil {
		task.Promise, _ = GetAction(task.PromiseAction.String())
	}
	if task.ExecutionAction != nil {
		task.Execution, _ = GetAction(task.ExecutionAction.String())
	}
	return task, nil
}

func (task *Task) CreateOrUpdate() error {
	if task.UUID.String() == "00000000-0000-0000-0000-000000000000" {
		// task was generated from json with an unknown UUID.  Fix up
		task.UUID = gocql.TimeUUID()
	}
	if task.ExecutionAction == nil || task.ExecutionAction.String() == "00000000-0000-0000-0000-000000000000" {
		// Cannot generate an action without a valid execution action
		return errors.New("Unspecified execution action")
	}
	if task.Queue == nil || task.Queue.String() == "00000000-0000-0000-0000-000000000000" {
		// Unspecified queue.  Assign to the "root" queue which is defined in schema.cql
		uuid, _ := gocql.ParseUUID("11111111-1111-1111-1111-111111111111")
		task.Queue = &uuid
	}
	if task.Status == "" {
		// New tasks need their status set...
		task.Status = TaskPending
	}
	q, err := GetQueue(task.Queue.String())
	if err == nil {
		if task.When.IsZero() && q.QueueType == QueueAsync {
			return errors.New("No timestamp set for async queue")
		}
		if task.When.IsZero() {
			// fix up the timestamp before embedding into the DB.  Go defaults to an epoch of 1754, Cassandra only supports 1970 onwards
			task.When = time.Date(1975, time.January, 0, 0, 0, 0, 0, time.UTC)
		}
		task.CreateOrUpdateTags()
		bind := cqlr.Bind(`insert into tasks (task_uuid, queue_uuid, execution_action, name, priority, promise_action, status, when) values (?, ?, ?, ?, ?, ?, ?, ?)`, task)
		if err := bind.Exec(session); err != nil {
			return err
		}
		return task.createOrUpdateInSubTables()
	} else {
		return errors.New("Unknown queue")
	}

}

func (task Task) createOrUpdateInSubTables() error {
	q, err := GetQueue(task.Queue.String())
	if err == nil {
		dQuery := session.Query(`delete from async_tasks where queue_uuid = ? and status = ? and when = ? and task_uuid = ?`, q.UUID, task.Status, task.When, task.UUID)
		iBind := cqlr.Bind(`insert into async_tasks (queue_uuid, status, when, task_uuid) values (?, ?, ?, ?)`, task)
		if q.QueueType == QueueSync {
			dQuery = session.Query(`delete from sync_tasks where queue_uuid = ? and status = ? and priority = ? and task_uuid = ?`, q.UUID, task.Status, task.Priority, task.UUID)
			iBind = cqlr.Bind(`insert into sync_tasks (queue_uuid, status, priority, task_uuid) values (?, ?, ?, ?)`, task)
		} else {

		}
		if err := dQuery.Exec(); err != nil {
			return err
		}
		if err := iBind.Exec(session); err != nil {
			return err
		}
	}
	return nil
}

func (task Task) Delete() error {
	task.DeleteTags()
	task.Status = TaskDeleted
	if err := session.Query(`update tasks set status = ? where task_uuid = ?`, task.Status, task.UUID).Exec(); err != nil {
		return err
	}
	return task.createOrUpdateInSubTables()
}

func (task Task) SetStatus(status string) error {
	task.Status = status
	if err := session.Query(`update tasks set status = ? where queue_uuid = ? and task_uuid = ? and when = ?`, task.Status, task.Queue, task.UUID, task.When).Exec(); err != nil {
		return err
	}
	return nil
}

func (t *Task) LoadTags() {
	t.OurTags = GetTagsForObject(t.UUID)
}

func (t Task) CreateOrUpdateTags() {
	SetTagsForObject(t.UUID, t.OurTags, "task")
}

func (t Task) DeleteTags() {
	DeleteTagsForObject(t.UUID)
}

func (t Task) Execute(sync bool) {
	log.WithFields(log.Fields{"task": t.UUID}).Info("Executing Task")
	successes := 0
	count := 0
	if t.ExecutionAction != nil && t.ExecutionAction.String() != "00000000-0000-0000-0000-000000000000" {
		count++
		if t.Execution.ExecuteAction(sync) {
			successes++
		}
	}
	// only execute promise at this point if the sync type is async, e.g. we are not waiting on completion message
	// TODO - how to execute promise when completion message is returned?
	if t.PromiseAction != nil && t.PromiseAction.String() != "00000000-0000-0000-0000-000000000000" && sync == false {
		count++
		if t.Promise.ExecuteAction(sync) {
			successes++
		}
	}
	if successes == 0 {
		t.Status = TaskFailed
	} else if count != successes {
		t.Status = TaskPartiallyFailed
	} else {
		t.Status = TaskComplete
	}
	t.CreateOrUpdate()
}
