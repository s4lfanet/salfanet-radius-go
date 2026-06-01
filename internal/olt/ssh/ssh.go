// Package ssh provides SSH-based CLI session management for ZTE OLTs.
//
// API mirrors the telnet package (Execute, ExecuteMultiple) so callers can
// switch between Telnet and SSH transparently via the zte.CLIPool interface.
//
// A fresh SSH connection is opened for each ExecuteMultiple call to avoid
// stale-connection problems; ZTE CLI sessions are short-lived so the overhead
// is acceptable.
package ssh

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	xssh "golang.org/x/crypto/ssh"
)

// Config holds SSH connection parameters for a ZTE OLT.
type Config struct {
	Host           string
	Port           int
	Username       string
	Password       string
	Prompt         string        // CLI prompt suffix (e.g. "#" for ZTE)
	CommandTimeout time.Duration // timeout per command (wait for next prompt)
	DialTimeout    time.Duration // TCP + SSH handshake timeout
}

// DefaultConfig returns sensible defaults for ZTE C320 SSH CLI.
func DefaultConfig(host string, port int, username, password string) Config {
	if port == 0 {
		port = 22
	}
	return Config{
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       password,
		Prompt:         "#",
		CommandTimeout: 30 * time.Second,
		DialTimeout:    15 * time.Second,
	}
}

// Pool provides SSH CLI command execution for ZTE OLTs.
// Despite the name it is stateless — each Execute/ExecuteMultiple call opens
// and closes its own SSH connection.  Included for interface compatibility
// with telnet.Pool.
type Pool struct {
	cfg Config
}

// New creates a new Pool with the given configuration.
func New(cfg Config) *Pool {
	return &Pool{cfg: cfg}
}

// Close is a no-op (stateless pool, no persistent connections to release).
func (p *Pool) Close() {}

// Execute opens an SSH connection, runs cmd in an interactive shell, returns output.
func (p *Pool) Execute(cmd string) (string, error) {
	return p.ExecuteMultiple([]string{cmd})
}

// ExecuteMultiple opens one SSH connection, runs all commands in a single
// interactive shell session, and returns the combined output.
func (p *Pool) ExecuteMultiple(cmds []string) (string, error) {
	client, err := p.dial()
	if err != nil {
		return "", err
	}
	defer client.Close()
	return runShellSession(client, p.cfg, cmds)
}

func (p *Pool) dial() (*xssh.Client, error) {
	cfg := &xssh.ClientConfig{
		User: p.cfg.Username,
		Auth: []xssh.AuthMethod{
			xssh.Password(p.cfg.Password),
			// keyboard-interactive as fallback — some ZTE firmware variants use this
			xssh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = p.cfg.Password
				}
				return answers, nil
			}),
		},
		// OLTs use self-signed / untrusted host keys; strict host key checking is
		// not practical without a PKI deployment in an ISP NMS context.
		HostKeyCallback: xssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         p.cfg.DialTimeout,
	}
	addr := net.JoinHostPort(p.cfg.Host, fmt.Sprintf("%d", p.cfg.Port))
	client, err := xssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return client, nil
}

// runShellSession opens an interactive PTY shell, executes commands one by one,
// and returns the combined output (login banner is discarded; prompts are stripped).
func runShellSession(client *xssh.Client, cfg Config, cmds []string) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh new session: %w", err)
	}
	defer sess.Close()

	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("ssh stdin pipe: %w", err)
	}

	stdoutPipe, err := sess.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("ssh stdout pipe: %w", err)
	}

	// Request a PTY — required by some ZTE firmware for interactive CLI.
	// ECHO=0 suppresses command echo so only command output is in the stream.
	_ = sess.RequestPty("xterm", 200, 200, xssh.TerminalModes{
		xssh.ECHO:          0,
		xssh.TTY_OP_ISPEED: 115200,
		xssh.TTY_OP_OSPEED: 115200,
	})

	if err := sess.Shell(); err != nil {
		return "", fmt.Errorf("ssh shell: %w", err)
	}

	reader := bufio.NewReader(stdoutPipe)

	// Drain login banner / MOTD and wait for the first CLI prompt.
	if _, err := readUntilPrompt(reader, cfg.Prompt, cfg.DialTimeout); err != nil {
		return "", fmt.Errorf("ssh initial prompt: %w", err)
	}

	var combined strings.Builder
	for _, cmd := range cmds {
		if _, err := fmt.Fprintf(stdinPipe, "%s\n", cmd); err != nil {
			return combined.String(), fmt.Errorf("ssh write %q: %w", cmd, err)
		}
		out, err := readUntilPrompt(reader, cfg.Prompt, cfg.CommandTimeout)
		combined.WriteString(out)
		if err != nil {
			return combined.String(), fmt.Errorf("ssh prompt timeout after %q: %w", cmd, err)
		}
	}

	_, _ = fmt.Fprintf(stdinPipe, "exit\n")
	_ = stdinPipe.Close()
	return combined.String(), nil
}

// readUntilPrompt reads bytes from r until prompt appears at the end of the
// accumulated buffer, or until timeout expires.
// A goroutine performs the blocking reads; the outer select enforces the timeout.
// When the session closes (EOF) the goroutine exits cleanly.
func readUntilPrompt(r *bufio.Reader, prompt string, timeout time.Duration) (string, error) {
	type result struct {
		out string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var buf strings.Builder
		for {
			b, err := r.ReadByte()
			if err != nil {
				if err == io.EOF {
					ch <- result{buf.String(), nil}
				} else {
					ch <- result{buf.String(), err}
				}
				return
			}
			buf.WriteByte(b)
			if strings.HasSuffix(buf.String(), prompt) {
				ch <- result{buf.String(), nil}
				return
			}
		}
	}()
	select {
	case res := <-ch:
		return res.out, res.err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for CLI prompt %q", prompt)
	}
}
