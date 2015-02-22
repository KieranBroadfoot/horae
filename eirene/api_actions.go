package eirene

import (
	"encoding/json"
	"github.com/kieranbroadfoot/horae/types"
	"net/http"
	"net/url"
)

// @Title actions
// @Description This endpoint will return actions known to Horae, either as an array or based on specific criteria (tag).
// @Accept  json
// @Param   tag     query    string     false        "Tag against which you wish to limit actions returned"
// @Success 200 {array}  types.Action
// @Failure 400 {object} types.Error
// @Resource /actions
// @Router /actions [get]
func getActions(w http.ResponseWriter, r *http.Request, toEunomia chan types.EunomiaRequest) {
	u, _ := url.Parse(r.URL.String())
	queryParams := u.Query()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if val, ok := queryParams["tag"]; ok {
		if err := json.NewEncoder(w).Encode(types.GetActionsByTag(val[0])); err != nil {
			panic(err)
		}
	} else {
		if err := json.NewEncoder(w).Encode(types.GetActions()); err != nil {
			panic(err)
		}
	}
}
