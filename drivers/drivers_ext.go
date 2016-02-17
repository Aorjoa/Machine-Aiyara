package drivers

import (
	"os/exec"

	"github.com/docker/machine/ssh"
)

func GetSSHCommandFromDriver(d Driver, args ...string) (*exec.Cmd, error) {
	if _, ok := d.(interface {
		GetSSHPasswd() string
	}); !ok {
		return getSSHCommandFromDriver(d, args...)
	}

	return getSSHCommandWithSSHPassFromDriver(d, args...)
}

func getSSHCommandWithSSHPassFromDriver(d Driver, args ...string) (*exec.Cmd, error) {
	host, err := d.GetSSHHostname()
	if err != nil {
		return nil, err
	}

	port, err := d.GetSSHPort()
	if err != nil {
		return nil, err
	}

	user := d.GetSSHUsername()
	passwd := ""
	// this is the hack to make it return a password rather than key path
	if d0, ok := d.(interface {
		GetSSHPasswd() string
	}); ok {
		passwd = d0.GetSSHPasswd()
		if passwd == "" {
			keyPath := d.GetSSHKeyPath()
			return ssh.GetSSHCommand(host, port, user, keyPath, args...), nil
		}
	}

	return ssh.GetSSHCommandWithSSHPass(host, port, user, passwd, args...), nil
}
