package provision

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/machine/drivers"
	"github.com/docker/machine/libmachine/auth"
	"github.com/docker/machine/libmachine/provision/pkgaction"
	"github.com/docker/machine/libmachine/swarm"
	"github.com/docker/machine/utils"
)

func init() {
	Register("Aiyara", &RegisteredProvisioner{
		New: NewAiyaraProvisioner,
	})
}

func NewAiyaraProvisioner(d drivers.Driver) Provisioner {
	return &AiyaraProvisioner{
		packages: []string{
			"curl",
		},
		Driver: d,
	}
}

type AiyaraProvisioner struct {
	packages      []string
	OsReleaseInfo *OsRelease
	Driver        drivers.Driver
	SwarmOptions  swarm.SwarmOptions
}

func (provisioner *AiyaraProvisioner) Service(name string, action pkgaction.ServiceAction) error {
	command := fmt.Sprintf("service %s %s", name, action.String())

	cmd, err := provisioner.SSHCommand(command)
	if err != nil {
		return err
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (provisioner *AiyaraProvisioner) Package(name string, action pkgaction.PackageAction) error {
	log.Debug("Package doing nothing")
	return nil
}

func (provisioner *AiyaraProvisioner) dockerDaemonResponding() bool {
	cmd, err := provisioner.SSHCommand("docker version")
	if err != nil {
		log.Warn("Error getting SSH command to check if the daemon is up: %s", err)
		return false
	}
	if err := cmd.Run(); err != nil {
		log.Debug("Error checking for daemon up: %s", err)
		return false
	}

	// The daemon is up if the command worked.  Carry on.
	return true
}

func (provisioner *AiyaraProvisioner) installPublicKey() error {

	if cmd, err := provisioner.SSHCommand("mkdir ~/.ssh"); err == nil {
		// skip error
		cmd.Run()
	} else {
		return err
	}

	publicKey, err := ioutil.ReadFile(provisioner.Driver.GetSSHKeyPath() + ".pub")
	if err != nil {
		return err
	}

	if cmd, err := provisioner.SSHCommand(fmt.Sprintf("echo \"%s\" | tee -a ~/.ssh/authorized_keys", string(publicKey))); err == nil {
		log.Info("Install public key to server")
		return cmd.Run()
	} else {
		return err
	}
}

func (provisioner *AiyaraProvisioner) installCustomDocker() error {
	// if the old version running, stop it
	provisioner.Service("docker", pkgaction.Stop)

	if cmd, err := provisioner.SSHCommand("unlink /usr/bin/docker || mkdir -p /opt/docker || unlink /opt/docker/docker"); err == nil {
		// skip error
		cmd.Run()
	} else {
		return err
	}

	if cmd, err := provisioner.SSHCommand("wget --no-check-certificate -q -O/opt/docker/docker.tar.xz https://dl.dropboxusercontent.com/u/9350284/docker.tar.xz && (cd /opt/docker && tar -xf docker.tar.xz)"); err == nil {
		err = cmd.Run()
		if err != nil {
			return err
		}
	} else {
		return err
	}

	if cmd, err := provisioner.SSHCommand("chmod +x /opt/docker/docker && ln -s /opt/docker/docker /usr/bin/docker"); err == nil {
		err = cmd.Run()
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// install init.d script
	if cmd, err := provisioner.SSHCommand("wget --no-check-certificate -q -O/etc/init.d/docker https://dl.dropboxusercontent.com/u/9350284/initd-docker.txt && chmod +x /etc/init.d/docker"); err == nil {
		err = cmd.Run()
		if err != nil {
			return err
		}
	} else {
		return err
	}

	provisioner.Service("docker", pkgaction.Start)

	return nil
}

func (provisioner *AiyaraProvisioner) Provision(swarmOptions swarm.SwarmOptions, authOptions auth.AuthOptions) error {

	log.Debug("Entering Provision")

	if err := provisioner.SetHostname(provisioner.Driver.GetMachineName()); err != nil {
		return err
	}

	if err := provisioner.installPublicKey(); err != nil {
		return err
	}

	if d0, ok := provisioner.Driver.(interface {
		ClearSSHPasswd()
	}); ok {
		d0.ClearSSHPasswd()
	}

	if err := provisioner.installCustomDocker(); err != nil {
		return err
	}

	if err := utils.WaitFor(provisioner.dockerDaemonResponding); err != nil {
		return err
	}

	if err := ConfigureAuth(provisioner, authOptions); err != nil {
		return err
	}

	if err := configureSwarm(provisioner, swarmOptions); err != nil {
		return err
	}

	return nil
}

func (provisioner *AiyaraProvisioner) Hostname() (string, error) {
	cmd, err := provisioner.SSHCommand("hostname")
	if err != nil {
		return "", err
	}

	var so bytes.Buffer
	cmd.Stdout = &so

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return so.String(), nil
}

func (provisioner *AiyaraProvisioner) SetHostname(hostname string) error {
	cmd, err := provisioner.SSHCommand(fmt.Sprintf(
		"hostname %s && echo %q | tee /etc/hostname && echo \"127.0.0.1 %s\" | tee -a /etc/hosts",
		hostname,
		hostname,
		hostname,
	))

	if err != nil {
		return err
	}

	return cmd.Run()
}

func (provisioner *AiyaraProvisioner) GetDockerOptionsDir() string {
	return "/etc/docker"
}

func (provisioner *AiyaraProvisioner) SSHCommand(args ...string) (*exec.Cmd, error) {
	return drivers.GetSSHCommandFromDriver(provisioner.Driver, args...)
}

func (provisioner *AiyaraProvisioner) CompatibleWithHost() bool {
	id := provisioner.OsReleaseInfo.Id
	return id == "ubuntu" || id == "debian" || id=="aiyara"
}

func (provisioner *AiyaraProvisioner) SetOsReleaseInfo(info *OsRelease) {
	provisioner.OsReleaseInfo = info
}

func (provisioner *AiyaraProvisioner) GenerateDockerOptions(dockerPort int, authOptions auth.AuthOptions) (*DockerOptions, error) {
	defaultDaemonOpts := getDefaultDaemonOpts(provisioner.Driver.DriverName(), authOptions)
	aiyaraOpts := fmt.Sprintf("--label=architecture=%s", "arm")
	daemonOpts := fmt.Sprintf("--host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:%d", dockerPort)
	daemonOptsDir := "/etc/default/docker"
	opts := fmt.Sprintf("%s %s %s", defaultDaemonOpts, daemonOpts, aiyaraOpts)
	daemonCfg := fmt.Sprintf("export DOCKER_OPTS=\\\"%s\\\"", opts)
	return &DockerOptions{
		EngineOptions:     daemonCfg,
		EngineOptionsPath: daemonOptsDir,
	}, nil
}

func (provisioner *AiyaraProvisioner) GetDriver() drivers.Driver {
	return provisioner.Driver
}
