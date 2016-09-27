package thethingsnetwork

import (
	"os"
	"os/user"

	log "github.com/Sirupsen/logrus"
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

// logger for logrus
type logger struct{}

func (l logger) Debug(msg string) {
	log.Debug(msg)
}
func (l logger) Info(msg string) {
	log.Info(msg)
}
func (l logger) Warn(msg string) {
	log.Warn(msg)
}
func (l logger) Error(msg string) {
	log.Error(msg)
}
func (l logger) Fatal(msg string) {
	log.Fatal(msg)
}
func (l logger) Debugf(msg string, v ...interface{}) {
	log.Debugf(msg, v...)
}
func (l logger) Infof(msg string, v ...interface{}) {
	log.Infof(msg, v...)
}
func (l logger) Warnf(msg string, v ...interface{}) {
	log.Warnf(msg, v...)
}
func (l logger) Errorf(msg string, v ...interface{}) {
	log.Errorf(msg, v...)
}
func (l logger) Fatalf(msg string, v ...interface{}) {
	log.Fatalf(msg, v...)
}
