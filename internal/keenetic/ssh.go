package keenetic

import (
	"fmt"
	"regexp"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshConn struct {
	*ssh.Client
	conf ConnConfig
}

func newSSHConn(conf ConnConfig) (*sshConn, error) {
	c := &sshConn{Client: nil, conf: conf}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *sshConn) connect() error {
	timeout := c.conf.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	var (
		addr         = fmt.Sprintf("%s:%d", c.conf.Host, c.conf.Port)
		clientConfig = &ssh.ClientConfig{
			User: c.conf.User,
			Auth: []ssh.AuthMethod{
				ssh.Password(c.conf.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
			Timeout:         timeout,
		}
		err error
	)

	c.Client, err = ssh.Dial("tcp", addr, clientConfig)

	return err
}

func (c *sshConn) exec(cmd string) ([]byte, error) {
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
