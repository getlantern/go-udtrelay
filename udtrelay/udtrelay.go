// Package udtrelay provides a convenience wrapper API around the udt gateway
// system (http://sourceforge.net/projects/udtgate/).
package udtrelay

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"

	"github.com/hailiang/socks"
	"github.com/oxtoacart/byteexec"
)

type natty struct {
	cmd *exec.Cmd
}

// Server provides a UDT gateway server
type Server struct {
	Port     int
	PeerAddr string
	DebugOut io.Writer

	natty
}

// Client provides a UDT gateway socks client
type Client struct {
	SOCKSPort int
	Port      int
	PeerAddr  string
	DebugOut  io.Writer
	DialWith  func(net, addr string) (net.Conn, error)

	natty
}

// Run runs the server.  This function blocks until the server stops running.
// The server will automatically stop if the current program receives a SIGINT.
func (server *Server) Run() error {
	params := []string{"-S", "-N", "-N", "0", strconv.Itoa(server.Port), server.PeerAddr}
	return server.run(params, server.DebugOut)
}

// Run runs the client.  This function blocks until the client stops running.
// The client will automatically stop if the current program receives a SIGINT.
func (client *Client) Run() error {
	params := []string{"-C", strconv.Itoa(client.SOCKSPort), strconv.Itoa(client.Port), client.PeerAddr}
	client.DialWith = socks.DialSocksProxy(socks.SOCKS4, fmt.Sprintf("127.0.0.1:%d", client.SOCKSPort))
	return client.run(params, client.DebugOut)
}

// Stop stops this client or server.
func (natty *natty) Stop() {
	proc := natty.cmd.Process
	if proc != nil {
		proc.Signal(os.Interrupt)
	}
}

func (natty *natty) run(params []string, debugOut io.Writer) error {
	bytes, err := Asset("udtrelay")
	if err != nil {
		return err
	}
	be, err := byteexec.NewByteExec(bytes)
	if err != nil {
		return err
	}

	natty.cmd = be.Command(params...)
	out := debugOut
	if out == nil {
		out = ioutil.Discard
	}
	stdout, err := natty.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := natty.cmd.StderrPipe()
	if err != nil {
		return err
	}

	go io.Copy(out, stdout)
	go io.Copy(out, stderr)
	natty.stopOnSigINT()

	return natty.cmd.Run()
}

func (natty *natty) stopOnSigINT() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		natty.Stop()
	}()
}
