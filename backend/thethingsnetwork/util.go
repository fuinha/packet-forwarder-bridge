package thethingsnetwork

import (
	"os"
	"os/user"
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
