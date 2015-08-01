package irc

import (
	"fmt"

	"github.com/sorcix/irc"
)

// CommandHandler allows objects implementing this interface to be registered to serve a particular IRC command
type CommandHandler interface {
	ServeIRC(message *irc.Message, client *Client)
}

// CommandHandlerFunc is a wrapper to to use regular functions as a CommandHandler
type CommandHandlerFunc func(message *irc.Message, client *Client)

// ServeIRC services a given IRC message from the given client
func (f CommandHandlerFunc) ServeIRC(message *irc.Message, client *Client) {
	f(message, client)
}

// PingHandler is a CommandHandler to respond to IRC PING commands from a client
// Implemented according to RFC 1459 4.6.2 and RFC 2812 3.7.2
func PingHandler(message *irc.Message, client *Client) {
	client.Pong()

}

// PongHandler is a CommandHandler to respond to IRC PONG commands from a client
// Implemented according to RFC 1459 4.6.3 and RFC 2812 3.7.3
func PongHandler(message *irc.Message, client *Client) {
	//client.Ping()

}

// QuitHandler is a CommandHandler to respond to IRC QUIT commands from a client
// Implemented according to RFC 1459 4.1.6 and RFC 2812 3.1.7
func QuitHandler(message *irc.Message, client *Client) {

	m := irc.Message{Prefix: client.server.Prefix, Command: irc.ERROR, Trailing: "quit"}

	client.Encode(&m)
	client.Close()
}

// NickHandler is a CommandHandler to respond to IRC NICK commands from a client
// Implemented according to RFC 1459 4.1.2 and RFC 2812 3.1.2
func NickHandler(message *irc.Message, client *Client) {

	var m irc.Message
	name := client.server.config.Name
	nickname := client.Nickname

	if len(message.Params) == 0 {
		m = irc.Message{Prefix: &irc.Prefix{Name: name}, Command: irc.ERR_NONICKNAMEGIVEN, Trailing: "No nickname given"}
		client.Encode(&m)
		return
	}

	newNickname := message.Params[0]

	_, found := client.server.ClientsByNick[newNickname]

	switch {
	case !client.authorized:
		m = irc.Message{Prefix: &irc.Prefix{Name: name}, Command: irc.ERR_PASSWDMISMATCH, Params: []string{newNickname}, Trailing: "Password incorrect"}

	case found: // nickname already in use
		fmt.Println("Nickname already used")
		m = irc.Message{Prefix: &irc.Prefix{Name: name}, Command: irc.ERR_NICKNAMEINUSE, Params: []string{newNickname}, Trailing: "Nickname is already in use"}

	default:
		if len(client.Nickname) == 0 && len(client.Username) != 0 { // Client is connected now, show MOTD ...
			client.Nickname = newNickname
			client.server.AddClientNick(client)
			client.Welcome()
		} else { //change client name
			client.Nickname = newNickname
			client.server.UpdateClientNick(client, nickname)
			//fmt.Println("Updating client name")
		}
	}

	if len(m.Command) != 0 {
		client.Encode(&m)
	}

}

// UserHandler is a CommandHandler to respond to IRC USER commands from a client
// Implemented according to RFC 1459 4.1.3 and RFC 2812 3.1.3
func UserHandler(message *irc.Message, client *Client) {
	var m irc.Message
	serverName := client.server.config.Name
	//nickname := client.Nickname

	if len(client.Username) != 0 { // Already registered
		m = irc.Message{Prefix: &irc.Prefix{Name: serverName}, Command: irc.ERR_ALREADYREGISTRED, Trailing: "You may not reregister"}
		client.Encode(&m)
		return
	}

	if len(message.Params) != 3 {
		m = irc.Message{Prefix: &irc.Prefix{Name: serverName}, Command: irc.ERR_NEEDMOREPARAMS, Trailing: "Not enough parameters"}
		client.Encode(&m)
		return
	}

	name := message.Params[0]
	username := message.Params[1]
	hostname := message.Params[2]
	realName := message.Trailing

	client.Name = name
	client.Username = username
	client.Host = hostname
	client.RealName = realName
	if len(m.Command) == 0 && len(client.Nickname) != 0 { // Client has finished connecting
		client.Welcome()
		return
	}

	if len(m.Command) != 0 {
		client.Encode(&m)
	}

}
