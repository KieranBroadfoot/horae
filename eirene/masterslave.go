package eirene

import (
	"net/http"
)

// Eirene middleware. Handles behaviour for nodes which are neither ready for action or acting as slaves

type MasterSlave struct {
	available  bool
	master     bool
	masterAddr string
	masterPort string
}

func NewMasterSlave() *MasterSlave {
	return &MasterSlave{false, false, "horae.co", "80"}
}

func (m *MasterSlave) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if m.available != true {
		// return 503
		rw.WriteHeader(http.StatusServiceUnavailable)
	} else if m.master != true {
		// return 301 redirect to current master node
		http.Redirect(rw, r, m.masterAddr+":"+m.masterPort+r.URL.Path, 301)
	} else {
		next(rw, r)
	}
}

func (m *MasterSlave) setAvailableAsMaster(addr string, port string) {
	m.available = true
	m.master = true
	m.masterAddr = addr
	m.masterPort = port
}

func (m *MasterSlave) setAvailableAsSlave(addr string, port string) {
	m.available = true
	m.master = false
	m.masterAddr = addr
	m.masterPort = port
}

func (m *MasterSlave) setUnavailable() {
	m.available = false
	m.master = false
}

func (m *MasterSlave) isMaster() bool {
	return m.master
}

func (m *MasterSlave) currentMaster() (string, string) {
	return m.masterAddr, m.masterPort
}

func (m *MasterSlave) currentMasterAsURI() (string) {
	return "http://"+m.masterAddr+":"+m.masterPort+"/"
}
