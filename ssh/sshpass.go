package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func GetSSHCommandWithSSHPass(host string, port int, user string, passwd string, args ...string) *exec.Cmd {
	defaultSSHArgs := []string{
		fmt.Sprintf("-p%s", passwd),
		"ssh",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=no", // don't bother checking in ~/.ssh/known_hosts
		"-o", "UserKnownHostsFile=/dev/null", // don't write anything to ~/.ssh/known_hosts
		"-o", "ConnectionAttempts=30", // retry 30 times if SSH connection fails
		"-o", "LogLevel=quiet", // suppress "Warning: Permanently added '[localhost]:2022' (ECDSA) to the list of known hosts."
		"-p", fmt.Sprintf("%d", port),
		fmt.Sprintf("%s@%s", user, host),
	}

	sshArgs := append(defaultSSHArgs, args...)
	cmd := exec.Command("sshpass", sshArgs...)
	cmd.Stderr = os.Stderr

	if os.Getenv("DEBUG") != "" {
		cmd.Stdout = os.Stdout
	}

	log.Debugf("executing: %v", strings.Join(cmd.Args, " "))

	return cmd
}
