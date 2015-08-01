package irc

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/sorcix/irc"
)

// Client represents an IRC Client connection to the server
type Client struct {
	*irc.Conn
	conn     net.Conn
	Nickname string
	Name     string
	Host     string
	Username string
	RealName string

	Prefix *irc.Prefix

	server     *Server
	authorized bool

	idleTimer *time.Timer
	quitTimer *time.Timer

	awayMessage string
}

func (s *Server) newClient(ircConn *irc.Conn, conn net.Conn) *Client {
	client := &Client{Conn: ircConn, conn: conn, server: s}
	client.authorized = len(s.config.Password) == 0
	client.idleTimer = time.AfterFunc(time.Minute, client.idle)
	return client
}

// Close cleans up the IRC client and closes the connection
func (c *Client) Close() error {
	c.server.RemoveClient(c)
	c.server.RemoveClientNick(c)

	return c.Conn.Close()
}

// Ping sends an IRC PING command to a client
func (c *Client) Ping() {
	m := irc.Message{Command: irc.PING, Params: []string{"JuddBot"}, Trailing: "JuddBot"}
	c.Encode(&m)
}

// Pong sends an IRC PONG command to the client
func (c *Client) Pong() {
	m := irc.Message{Command: irc.PONG, Params: []string{"JuddBot"}, Trailing: "JuddBot"}
	c.Encode(&m)
}

func (c *Client) handleIncoming() {
	c.server.AddClient(c)
	for {
		message, err := c.Decode()
		if err != nil || message == nil {

			_, closedError := err.(*net.OpError)
			if err == io.EOF || err == io.ErrClosedPipe || closedError || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			println("Error decoding incoming message", err.Error())
			return
			//continue
		}
		//println(message.String())

		c.idleTimer.Stop()
		c.idleTimer = time.AfterFunc(time.Minute, c.idle)
		if c.quitTimer != nil {
			c.quitTimer.Stop()
			c.quitTimer = nil
		}

		c.server.CommandsMux.ServeIRC(message, c)

	}

}

func (c *Client) idle() {
	c.Ping()
	c.quitTimer = time.AfterFunc(time.Minute, c.quit)
}

func (c *Client) quit() {
	c.Quit()
}

// Quit sends the IRC Quit command and closes the connection
func (c *Client) Quit() {
	m := irc.Message{Prefix: &irc.Prefix{Name: c.server.config.Name}, Command: irc.QUIT,
		Params: []string{c.Nickname}}

	c.Encode(&m)
	c.Close()
}

// Welcome handles initial client connection IRC protocols for a client.
// Welcome procedure includes IRC WELCOME, Host Info, and MOTD
func (c *Client) Welcome() {

	// Have all client info now
	c.Prefix = &irc.Prefix{Name: c.Nickname, User: c.Username, Host: c.Host}

	m := irc.Message{Prefix: c.server.Prefix, Command: irc.RPL_WELCOME,
		Params: []string{c.Nickname, c.server.config.Welcome}}

	err := c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.server.Prefix, Command: irc.RPL_YOURHOST,
		Params: []string{c.Nickname, fmt.Sprintf("Your host is %s", c.server.config.Name)}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.server.Prefix, Command: irc.RPL_CREATED,
		Params: []string{c.Nickname, fmt.Sprintf("This server was created %s", c.server.created)}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.server.Prefix, Command: irc.RPL_MYINFO,
		Params: []string{c.Nickname, fmt.Sprintf("%s  - Golang IRC server", c.server.config.Name)}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.server.Prefix, Command: irc.RPL_MOTDSTART,
		Params: []string{c.Nickname, fmt.Sprintf("%s  - Message of the day", c.server.config.Name)}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.server.Prefix, Command: irc.RPL_MOTD,
		Params: []string{c.Nickname, c.server.config.MOTD}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.server.Prefix, Command: irc.RPL_ENDOFMOTD,
		Params: []string{c.Nickname, "End of MOTD"}}

	err = c.Encode(&m)
	if err != nil {
		return
	}
}
