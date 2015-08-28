package irc

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
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

	Server     *Server
	Authorized bool
	Registered bool

	idleTimer *time.Timer
	quitTimer *time.Timer

	AwayMessage string

	channels     map[string]*Channel
	channelMutex sync.RWMutex

	*UserModeSet
}

func (s *Server) newClient(ircConn *irc.Conn, conn net.Conn) *Client {
	client := &Client{Conn: ircConn, conn: conn, Server: s}
	client.Authorized = len(s.Config.Password) == 0
	client.idleTimer = time.AfterFunc(time.Minute*1, client.quit)
	client.channels = map[string]*Channel{}
	client.UserModeSet = NewUserModeSet()
	return client
}

// Close cleans up the IRC client and closes the connection
func (c *Client) Close() error {
	c.Server.RemoveClient(c)
	c.Server.RemoveClientNick(c)

	return c.Conn.Close()
}

// Ping sends an IRC PING command to a client
func (c *Client) Ping() {
	m := irc.Message{Command: irc.PING, Trailing: c.Server.Config.Name}
	c.Encode(&m)
}

// Pong sends an IRC PONG command to the client
func (c *Client) Pong() {
	m := irc.Message{Command: irc.PONG, Trailing: c.Server.Config.Name}
	c.Encode(&m)
}

func (c *Client) handleIncoming() {
	c.Server.AddClient(c)
	for {
		message, err := c.Decode()
		if err != nil {

			_, closedError := err.(*net.OpError)
			if err == io.EOF || err == io.ErrClosedPipe || closedError || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}

			continue
		}
		if message == nil || message.Len() == 0 {
			continue
		}

		c.idleTimer.Stop()

		if !c.Registered { // if client isn't registered don't bother with PINGs
			c.idleTimer = time.AfterFunc(time.Minute*1, c.quit)
		} else {
			c.idleTimer = time.AfterFunc(time.Minute*3, c.idle)
		}

		if c.quitTimer != nil {
			c.quitTimer.Stop()
			c.quitTimer = nil
		}

		c.Server.CommandsMux.ServeIRC(message, c)

	}

}

func (c *Client) idle() {
	c.Ping()
	c.quitTimer = time.AfterFunc(time.Minute*3, c.quit)
}

func (c *Client) quit() {
	// Have client leave/part each channel
	for _, channel := range c.GetChannels() {
		channel.Quit(c, "Disconnected")
	}
	c.Quit()
}

// Quit sends the IRC Quit command and closes the connection
func (c *Client) Quit() {
	m := irc.Message{Prefix: &irc.Prefix{Name: c.Server.Config.Name}, Command: irc.QUIT,
		Params: []string{c.Nickname}}

	c.Encode(&m)
	c.Close()
}

// Welcome handles initial client connection IRC protocols for a client.
// Welcome procedure includes IRC WELCOME, Host Info, and MOTD
func (c *Client) Welcome() {

	// Have all client info now
	c.Prefix = &irc.Prefix{Name: c.Nickname, User: c.Name, Host: c.Host}
	c.Registered = true

	m := irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_WELCOME,
		Params: []string{c.Nickname, "Welcome to the Internet Relay Network", c.Prefix.String()}}

	err := c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_YOURHOST,
		Params: []string{c.Nickname, fmt.Sprintf("Your host is %s, running version %s", c.Server.Config.Name, c.Server.Config.Version)}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_CREATED,
		Params: []string{c.Nickname, fmt.Sprintf("This server was created %s", c.Server.created)}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_MYINFO,
		Params: []string{c.Nickname, fmt.Sprintf("%s  - Golang IRC server", c.Server.Config.Name)}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	// Send MOTD
	c.MOTD()

}

// MOTD returns the Message of the Day of the server to the client
func (c *Client) MOTD() {

	if len(c.Server.Config.MOTD) == 0 {
		m := irc.Message{Prefix: c.Server.Prefix, Command: irc.ERR_NOMOTD, Params: []string{c.Nickname}, Trailing: "MOTD File is missing"}
		c.Encode(&m)
	}

	m := irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_MOTDSTART,
		Params: []string{c.Nickname, fmt.Sprintf("%s  - Message of the day", c.Server.Config.Name)}}

	err := c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_MOTD,
		Params: []string{c.Nickname, c.Server.Config.MOTD}}

	err = c.Encode(&m)
	if err != nil {
		return
	}

	m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_ENDOFMOTD,
		Params: []string{c.Nickname, "End of MOTD"}}

	err = c.Encode(&m)
	if err != nil {
		return
	}
}

// AddChannel adds a channel to the client's active list
func (c *Client) AddChannel(channel *Channel) {
	c.channelMutex.Lock()
	defer c.channelMutex.Unlock()
	c.channels[channel.Name] = channel
}

// RemoveChannel removes a channel to the client's active list
func (c *Client) RemoveChannel(channel *Channel) {
	c.channelMutex.Lock()
	defer c.channelMutex.Unlock()
	delete(c.channels, channel.Name)
}

// GetChannels gets a list of channels this client is joined to
func (c *Client) GetChannels() map[string]*Channel {
	return c.channels
}

// UpdateNick updates the clients nicknamae to a new nickname
func (c *Client) UpdateNick(newNick string) {
	oldNick := c.Nickname
	c.Nickname = newNick

	c.Server.UpdateClientNick(c, oldNick)
	c.channelMutex.RLock()
	defer c.channelMutex.RUnlock()

	// Notify all people that should know (people on channels with this client)
	m := irc.Message{Prefix: c.Prefix, Command: irc.NICK, Trailing: c.Nickname}
	notified := map[string]interface{}{} // Just notify people once
	c.Encode(&m)
	notified[c.Nickname] = nil

	for _, channel := range c.channels {

		channel.UpdateMemberNick(c, oldNick)

		for client := range channel.members {
			cl, ok := c.Server.GetClientByNick(client)
			_, alreadyNotified := notified[client]
			if ok && !alreadyNotified {
				cl.Encode(&m)
				notified[client] = nil
			}
		}
	}

	c.Prefix.Name = newNick
}

// GetVisible returns a map of clients visible to this client
func (c *Client) GetVisible() map[string]*Client {
	clients := map[string]*Client{}
	for name, client := range c.Server.clientsByNick {
		if client.HasMode(UserModeInvisible) {
			continue
		}
		clients[name] = client
	}
	for _, channel := range c.channels {
		for member := range channel.members {
			tmp, ok := c.Server.GetClientByNick(member)
			if ok {
				clients[member] = tmp
			}
		}
	}
	return clients
}

// SendMessagetoVisible sends a message to all other visible clients
func (c *Client) SendMessagetoVisible(m *irc.Message) {
	for _, client := range c.GetVisible() {
		client.Encode(m)
	}
}

// Who rmanages responding to the WHO request for all visible clients of this client
func (c *Client) Who() {
	clients := map[string]*Client{}
	for name, client := range c.Server.clientsByNick {
		if client.HasMode(UserModeInvisible) {
			continue
		}
		msg := whoLine(client, nil, c.Nickname)
		m := irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_WHOREPLY, Params: strings.Fields(msg)}
		c.Encode(&m)

		clients[name] = client
	}
	for _, channel := range c.channels {
		for member := range channel.members {
			tmp, ok := c.Server.GetClientByNick(member)
			_, alreadySent := clients[member]
			if ok && !alreadySent {
				msg := whoLine(tmp, channel, c.Nickname)
				m := irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_WHOREPLY, Params: strings.Fields(msg)}
				c.Encode(&m)
				clients[member] = tmp
			}
		}
	}
	m := irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_ENDOFWHO, Params: []string{c.Nickname, "*"}, Trailing: "End of WHO list"}
	c.Encode(&m)
}

// MakeOper makes this client a server operator
func (c *Client) MakeOper() {
	c.AddMode(UserModeOperator)
	m := irc.Message{Prefix: c.Server.Prefix, Command: irc.MODE, Params: []string{c.Nickname, "+o"}}
	for name, client := range c.Server.clientsByNick {
		if name == c.Nickname {
			continue
		}
		client.Encode(&m)

	}
	m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_YOUREOPER, Params: []string{c.Nickname}, Trailing: "You are now an IRC operator"}
	c.Encode(&m)
}
