package libmachine

import (
	"os/exec"

	"github.com/docker/machine/ssh"
)

func (h *Host) GetSSHCommand(args ...string) (*exec.Cmd, error) {
	if driver, ok := h.Driver.(interface {
		GetSSHPasswd() string
	}); ok {
		addr, err := h.Driver.GetSSHHostname()
		if err != nil {
			return nil, err
		}

		user := h.Driver.GetSSHUsername()

		port, err := h.Driver.GetSSHPort()
		if err != nil {
			return nil, err
		}

		passwd := driver.GetSSHPasswd()

		// if password got cleared, use publickey
		if passwd == "" {
			return h.getSSHCommand(args...)
		}

		cmd := ssh.GetSSHCommandWithSSHPass(addr, port, user, passwd, args...)
		return cmd, nil
	}

	return h.getSSHCommand(args...)
}
