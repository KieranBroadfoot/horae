package eirene

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/kieranbroadfoot/horae/types"
	"io"
	"net/http"
)

// @Title queryqueue
// @Description Provides details of the requested queue including availability windows, type, associated tags and paths.
// @Accept  json
// @Param   uuid     path    string     false        "UUID of the requested queue"
// @Success 200 {object} types.Queue
// @Failure 400 {object} types.Error
// @Resource /queues
// @Router /queue/{uuid} [get]
func getQueue(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	vars := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(types.GetQueue(vars["uuid"])); err != nil {
		panic(err)
	}
}

// @Title createqueue
// @Description This endpoint enables the creation of a new queue.  All queues must be defined with a unique name and window of operation and type.  Optionally you may also define a series of tags to help searching for a particular queue or queues.  The queue type is either "sync" or "async".  If defined as "async" then any tasks available in the queue will be executed in the next availability window.  However, sync queues will execute tasks in a FIFO manner during the availability window.  To enable this, tasks associated to the queue must execute a task completion call when finished to ensure Horae can continue execution.  Optionally sync queues may also define a backpressure URI, operation, payload AND definition.  If Horae starts to see the queue meet the backpressure definition the callback will be executed.
// @Accept  json
// @Param   queue     query    types.Queue     true        "A queue object"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /queues
// @Router /queue [put]
func createQueue(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}

// @Title updatequeue
// @Description A queue may be updated via this endpoint.  The name, window of operation, and backpressure configuration.  If the window of operation is changed whilst it is active those tasks in-flight will continue but any others will be held back until the next window of operation.  Queues cannot change their "type" from sync to async or vice-versa.  You would need to delete and recreate because you would need to define the draining behaviour and existing tasks may not be aware of the need to callback on completion (when moving from async to sync).
// @Accept  json
// @Param   uuid     path    string     true        "UUID for updated queue"
// @Param	queue	 query	types.Queue true		"A queue object"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /queues
// @Router /queue/{uuid} [put]
func updateQueue(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}

// @Title deletequeue
// @Description When called a defined queue will either be immediately removed and all associated tasks deleted OR if requested the queue will be defined as "drain-only" which will delete the queue when it is empty.
// @Accept  json
// @Param   uuid     	path    string     	true    "Tag against which you wish to limit queues returned"
// @Param 	shouldDrain	query	bool		false	"If empty to set to false the queue will be immediately deleted along with any associated tasks.  If set to true the queue will only be removed when the queue is empty.  No new tasks can be added to the queue once set."
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /queues
// @Router /queue/{uuid} [delete]
func deleteQueue(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}
