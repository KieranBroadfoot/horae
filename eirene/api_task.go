package eirene

import (
	"github.com/kieranbroadfoot/horae/types"
	"io"
	"net/http"
)

// @Title querytask
// @Description The task endpoint will return a known Task with the appropriate UUID.  UUIDs are created by Horae during creation.
// @Accept  json
// @Param   uuid     path    string     false        "UUID of the requested task"
// @Success 200 {object} types.Task
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task/{uuid} [get]
func getTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}

// @Title createtask
// @Description The endpoint defines a method to create a task within Horae.  The task must always provide the callback API and Operation to call when initiated.  It may also include an optional payload value (typically a json blob) to be sent to the executing service. It must also define EITHER a queue into which it should be placed or an execution time (in UTC).  If an execution time is requested the task is placed into the "default" queue.  Optionally a task may define a series of tags in order to aid in searching.
// @Accept  json
// @Param   task     query    types.Task     true        "A task object"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task [put]
func createTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}

// @Title updatetask
// @Description A task may ONLY update its callback API endpoint or execution time (if it exists within the default queue).  If the task needs to be moved between queues then both delete and create should be undertaken.  The task may require changes to its behaviour to meet the requisite queues behaviour.
// @Accept  json
// @Param   uuid     path   string     	true        "UUID for updated task"
// @Param	queue	 query	types.Task  true		"A task object"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task/{uuid} [put]
func updateTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}

// @Title deletetask
// @Description When a task is deleted it will be immediately removed unless it is currently in execution (via a sync queue).  In this case the task will continue to complete as expected.
// @Accept  json
// @Param   uuid     	path    string     	true    "UUID of the task to be deleted"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task/{uuid} [delete]
func deleteTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}

// @Title completetask
// @Description When a task is defined within a synchronous queue it is essential that it signals completion to Horae.  This endpoint provides that completion mechanism.
// @Accept  json
// @Param   uuid     	path    string     	true    "UUID of completing task"
// @Success 200 {object} types.Success
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /task/{uuid}/complete [get]
func completeTask(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	// TODO - consider how best to marshall tags and paths into resulting object
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}
