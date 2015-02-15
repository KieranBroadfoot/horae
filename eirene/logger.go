package eirene

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"net/http"
	"time"
)

type EireneLogger struct {
	*log.Logger
}

func NewEireneLogger() *EireneLogger {
	return &EireneLogger{log.New()}
}

func (l *EireneLogger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	l.WithFields(log.Fields{"Method": r.Method, "URL": r.URL.Path}).Info("Request Started")

	next(rw, r)

	// undertake a type assertion to a negroni ResponseWriter so we can access the status field via the helper function
	res := rw.(negroni.ResponseWriter)
	l.WithFields(log.Fields{"Status": res.Status(), "StatusText": http.StatusText(res.Status()), "Time": time.Since(start)}).Info("Request Completed")
}
