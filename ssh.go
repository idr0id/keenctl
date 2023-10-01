package main

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"regexp"
)

type SshClient struct {
	sshClient *ssh.Client
}

func (c *SshClient) exec(cmd string) string {
	log.Trace().Str("cmd", cmd).Msg("execute ssh command")

	session, err := c.sshClient.NewSession()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create session")
	}
	defer func() {
		if err := session.Close(); err != io.EOF {
			log.Warn().Err(err).Msg("failed to close session")
		}
	}()

	session.Wait()

	buf, err := session.Output(cmd)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to execute command")
	}

	return removeEscapeSequences(string(buf))
}

func DialSshWithPasswd(addr, user, passwd string) (*SshClient, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(passwd),
		},
		//HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		HostKeyCallback: ssh.HostKeyCallback(
			func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
		),
	}

	log.Trace().
		Str("addr", addr).
		Str("user", user).
		Str("password", passwd).
		Msg("connecting")

	return Dial("tcp", addr, config)
}

func Dial(network, addr string, config *ssh.ClientConfig) (*SshClient, error) {
	sshClient, err := ssh.Dial(network, addr, config)
	if err != nil {
		return nil, err
	}

	return &SshClient{sshClient: sshClient}, nil
}

func removeEscapeSequences(input string) string {
	escapeSequencesRegex := regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

	return escapeSequencesRegex.ReplaceAllString(input, "")
}
