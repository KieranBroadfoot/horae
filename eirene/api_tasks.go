package eirene

import (
	"github.com/kieranbroadfoot/horae/types"
	"io"
	"net/http"
)

// @Title tasks
// @Description This endpoint will return tasks known to Horae, either as an array or based on specific criteria (tag or queue).
// @Accept  json
// @Param   tag     query    string     false        "Tag against which you wish to limit tasks returned"
// @Param   queue   query    string     false        "UUID of queue to scope tasks returned"
// @Success 200 {array}  types.Task
// @Failure 400 {object} types.Error
// @Resource /tasks
// @Router /tasks [get]
func getTasks(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	io.WriteString(w, "Hello world!")
	//toEunomia <- "FOO"
}
