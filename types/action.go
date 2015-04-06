package types

import (
	"bytes"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"
	"github.com/relops/cqlr"
	"net/http"
	"time"
	"strings"
)

type Action struct {
	UUID      gocql.UUID `cql:"action_uuid" json:"uuid,required"`
	Operation string     `cql:"operation" json:"operation,omitempty"`
	Payload   string     `cql:"payload" json:"payload,omitempty"`
	URI       string     `cql:"uri" json:"uri,omitempty"`
	Status    string     `cql:"status" json:"status,omitempty"`
	Failure   string     `cql:"failure" json:"failure,omitempty"`
	OurTags   []string   `json:"tags,omitempty" description:"Tags assigned to the action."`
}

func GetActions() []Action {
	query := session.Query("select * from actions")
	bind := cqlr.BindQuery(query)
	var action Action
	actions := []Action{}
	for bind.Scan(&action) {
		action.LoadTags()
		actions = append(actions, action)
	}
	return actions
}

func GetActionsByTag(tag string) []Action {
	var id gocql.UUID
	var action Action
	actions := []Action{}
	iteration := session.Query("select object_uuid from tags where type = 'action' and tag = ? allow filtering", tag).Iter()
	for iteration.Scan(&id) {
		q := session.Query("select * from actions where action_uuid = ? allow filtering", id)
		b := cqlr.BindQuery(q)
		b.Scan(&action)
		action.LoadTags()
		actions = append(actions, action)
	}
	return actions
}

func GetAction(actionUUID string) (Action, error) {
	query := session.Query("select * from actions where action_uuid = ?", actionUUID)
	bind := cqlr.BindQuery(query)
	var action Action
	if !bind.Scan(&action) {
		return Action{}, errors.New("Unknown action")
	}
	action.LoadTags()
	return action, nil
}

func (action *Action) CreateOrUpdate() error {
	if action.UUID.String() == "00000000-0000-0000-0000-000000000000" {
		// action was generated from json with an unknown UUID.  Fix up
		action.UUID = gocql.TimeUUID()
	}
	bind := cqlr.Bind(`insert into actions (action_uuid, operation, uri, payload, status, failure) values (?, ?, ?, ?, ?, ?)`, action)
	if err := bind.Exec(session); err != nil {
		return err
	} else {
		return nil
	}
}

func (action *Action) Delete() error {
	action.DeleteTags()
	bind := cqlr.Bind(`delete from actions where action_uuid = ?`, action)
	if err := bind.Exec(session); err != nil {
		log.Print("received error from delete")
		return err
	} else {
		return nil
	}
}

func (a *Action) LoadTags() {
	a.OurTags = GetTagsForObject(a.UUID)
}

func (a Action) CreateOrUpdateTags() {
	SetTagsForObject(a.UUID, a.OurTags, "action")
}

func (a Action) DeleteTags() {
	DeleteTagsForObject(a.UUID)
}

func (action *Action) Execute(task *Task) bool {
	start := time.Now()
	// create temp vars for uri and payload.  we don't want to save the resolved versions back to the DB
	uri := action.URI
	payload := action.Payload
	configMap := map[string]string{
		"<<HORAE_API_URI>>": Configuration.MasterURI,
		"<<HORAE_COMPLETION_URI>>": Configuration.MasterURI+"v1/task/"+task.UUID.String()+"/complete",
		"<<HORAE_TASK_UUID>>": task.UUID.String(),
		"<<HORAE_TASK_STATUS>>": task.Status,
	}
	for k, v := range configMap {
		uri = strings.Replace(uri, k, v, -1)
		payload = strings.Replace(payload, k, v, -1)
	}
	// log later so we have a resolved URI
	log.WithFields(log.Fields{"action": action.UUID, "URI": uri, "verb": action.Operation}).Info("Executing Action")
	response, error := action.makeRequest(uri, payload)
	if error != nil {
		action.Status = TaskFailed
		action.Failure = error.Error()
	} else {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			action.Status = TaskComplete
		} else {
			action.Status = TaskFailed
		}
	}
	action.CreateOrUpdate()
	log.WithFields(log.Fields{"action": action.UUID, "status": action.Status, "time": time.Since(start)}).Info("Finished Action Execution")
	if action.Status == TaskComplete {
		return true
	} else {
		return false
	}
}

func (a Action) makeRequest(uri string, payload string) (resp *http.Response, err error) {
	switch {
	case a.Operation == TaskGet:
		return http.Get(uri)
	case a.Operation == TaskPost:
		return http.Post(uri, "application/json", bytes.NewBufferString(payload))
	case a.Operation == TaskHead:
		return http.Head(uri)
	case a.Operation == TaskDelete:
		request, err := http.NewRequest(TaskDelete, uri, nil)
		if err != nil {
			return http.DefaultClient.Do(request)
		} else {
			return nil, err
		}
	}
	return nil, errors.New("No valided handler for " + a.Operation)
}
