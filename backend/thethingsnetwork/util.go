package thethingsnetwork

import (
	"os"
	"os/user"

	log "github.com/Sirupsen/logrus"
	"github.com/TheThingsNetwork/ttn/api"
	apex "github.com/apex/log"
)

// getID retrns the ID of this ttnctl
func getID() string {
	id := ""
	if user, err := user.Current(); err == nil {
		id += "-" + user.Username
	}
	if hostname, err := os.Hostname(); err == nil {
		id += "@" + hostname
	}
	return id
}

// WrapLogrus wraps logrus into an api.Logger
func WrapLogrus() api.Logger {
	return &logrusEntryWrapper{*log.NewEntry(log.StandardLogger())}
}

var _ api.Logger = &logrusEntryWrapper{}

type logrusEntryWrapper struct {
	log.Entry
}

func (w *logrusEntryWrapper) Debug(msg string) {
	w.Entry.Debug(msg)
}

func (w *logrusEntryWrapper) Info(msg string) {
	w.Entry.Info(msg)
}

func (w *logrusEntryWrapper) Warn(msg string) {
	w.Entry.Warn(msg)
}

func (w *logrusEntryWrapper) Error(msg string) {
	w.Entry.Error(msg)
}

func (w *logrusEntryWrapper) Fatal(msg string) {
	w.Entry.Fatal(msg)
}

func (w *logrusEntryWrapper) WithError(err error) api.Logger {
	return &logrusEntryWrapper{*w.Entry.WithError(err)}
}

func (w *logrusEntryWrapper) WithField(k string, v interface{}) api.Logger {
	return &logrusEntryWrapper{*w.Entry.WithField(k, v)}
}

func (w *logrusEntryWrapper) WithFields(fields apex.Fielder) api.Logger {
	return &logrusEntryWrapper{*w.Entry.WithFields(
		map[string]interface{}(fields.Fields()),
	)}
}
