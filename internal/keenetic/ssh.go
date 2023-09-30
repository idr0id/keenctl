package keenetic

import (
	"fmt"
	"regexp"
	"time"

	"golang.org/x/crypto/ssh"
)

type ConnConfig struct {
	Host                string        `koanf:"host"`
	Port                int           `koanf:"port"`
	User                string        `koanf:"user"`
	Password            string        `koanf:"password"`
	MaxParallelCommands uint          `koanf:"max_parallel_commands"`
	Timeout             time.Duration `koanf:"timeout"`
	DryRun              bool          `koanf:"dry_run"`
}

type sshClient struct {
	*ssh.Client
}

func newSSHClient(conf ConnConfig) (*sshClient, error) {
	timeout := conf.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	var (
		addr         = fmt.Sprintf("%s:%d", conf.Host, conf.Port)
		clientConfig = &ssh.ClientConfig{
			User: conf.User,
			Auth: []ssh.AuthMethod{
				ssh.Password(conf.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
			Timeout:         timeout,
		}
	)

	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, err
	}

	return &sshClient{client}, nil
}

func (c *sshClient) exec(cmd string) ([]byte, error) {
	session, err := c.NewSession()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = session.Close()
	}()

	buf, err := session.CombinedOutput(cmd)

	return stripEscapes(buf), err
}

var escapeSequencesRegex = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

func stripEscapes(input []byte) []byte {
	if len(input) == 0 {
		return nil
	}

	return escapeSequencesRegex.ReplaceAll(input, nil)
}
