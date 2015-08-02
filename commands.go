package irc

import (
	"fmt"
	"strings"

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

	var leavingMessage string
	if len(message.Params) != 0 {
		leavingMessage = message.Params[0]
	}
	for _, channel := range client.GetChannels() {
		channel.Quit(client, leavingMessage)
	}

	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERROR, Trailing: "quit"}

	client.Encode(&m)
	client.Close()
}

// NickHandler is a CommandHandler to respond to IRC NICK commands from a client
// Implemented according to RFC 1459 4.1.2 and RFC 2812 3.1.2
func NickHandler(message *irc.Message, client *Client) {

	var m irc.Message
	name := client.Server.Config.Name
	nickname := client.Nickname

	if len(message.Params) == 0 {
		m = irc.Message{Prefix: &irc.Prefix{Name: name}, Command: irc.ERR_NONICKNAMEGIVEN, Trailing: "No nickname given"}
		client.Encode(&m)
		return
	}

	newNickname := message.Params[0]

	_, found := client.Server.GetClientByNick(newNickname)

	switch {
	case !client.authorized:
		m = irc.Message{Prefix: &irc.Prefix{Name: name}, Command: irc.ERR_PASSWDMISMATCH, Params: []string{newNickname}, Trailing: "Password incorrect"}

	case found: // nickname already in use
		fmt.Println("Nickname already used")
		m = irc.Message{Prefix: &irc.Prefix{Name: name}, Command: irc.ERR_NICKNAMEINUSE, Params: []string{newNickname}, Trailing: "Nickname is already in use"}

	default:
		if len(client.Nickname) == 0 && len(client.Username) != 0 { // Client is connected now, show MOTD ...
			client.Nickname = newNickname
			client.Server.AddClientNick(client)
			client.Welcome()
		} else { //change client name
			client.Nickname = newNickname
			client.Server.UpdateClientNick(client, nickname)
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
	serverName := client.Server.Config.Name
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

// JoinHandler is a CommandHandler to respond to IRC JOIN commands from a client
// Implemented according to RFC 1459 4.2.1 and RFC 2812 3.2.1
func JoinHandler(message *irc.Message, client *Client) {
	channelNames := message.Params[0]
	if channelNames == "0" { // Leave all channels
		for _, channel := range client.GetChannels() {
			channel.Part(client, "")
		}
		return
	}
	channelList := strings.Split(channelNames, ",")
	keys := ""
	if len(message.Params) >= 2 {
		keys = message.Params[1]
	}
	keyList := strings.Split(keys, ",")

	for i, cName := range channelList {
		var key string
		if len(keyList) > i {
			key = keyList[i]
		}
		channel, ok := client.Server.GetChannel(cName)
		if !ok { // Channel doesn't exist  yet
			channel = NewChannel(client.Server, client)
			channel.Name = cName

			//channel.Members[client.Nickname] = client.Nickname

			channel.Key = key

			client.Server.AddChannel(channel)

		} else { //Channel already exists
		}
		//Notify channel members of new member
		channel.Join(client, key)
	}

}

// PartHandler is a CommandHandler to respond to IRC PART commands from a client
// Implemented according to RFC 1459 4.2.2 and RFC 2812 3.2.2
func PartHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Trailing: "Not enough parameters"}
		client.Encode(&m)
		return
	}

	for _, cName := range message.Params {
		channel, ok := client.Server.GetChannel(cName)
		if !ok { // Channel doesn't exist  yet

			m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOSUCHCHANNEL, Trailing: "You're not on that channel"}
			client.Encode(&m)
			continue

		} else { //Channel already exists
		}

		channel.Part(client, message.Trailing)
	}

}

// PrivMsgHandler is a CommandHandler to respond to IRC PRIVMSG commands from a client
// Implemented according to RFC 1459 4.4.1 and RFC 2812 3.3.1
func PrivMsgHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NORECIPIENT, Params: []string{client.Nickname}, Trailing: "No recipient given (PRIVMSG)"}
		client.Encode(&m)
		return
	}
	if len(message.Params) > 1 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_TOOMANYTARGETS, Params: []string{client.Nickname}}
		client.Encode(&m)
		return
	}
	if len(message.Trailing) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOTEXTTOSEND, Params: []string{client.Nickname}, Trailing: "No text to send"}
		client.Encode(&m)
		return
	}

	to := message.Params[0]
	ch, ok := client.Server.GetChannel(to)
	if ok { // message is to a channel
		ch.Message(client, message.Trailing)
		return

	}
	// message to a user?
	cl, ok := client.Server.GetClientByNick(to)
	if ok {
		m := irc.Message{Prefix: client.Prefix, Command: irc.PRIVMSG, Params: []string{cl.Nickname}, Trailing: message.Trailing}
		cl.Encode(&m)
		return
	}

	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOSUCHNICK, Params: []string{client.Nickname}, Trailing: "No recipient given (PRIVMSG)"}
	client.Encode(&m)

}

// NoticeHandler is a CommandHandler to respond to IRC NOTICE commands from a client
// Implemented according to RFC 1459 4.4.2 and RFC 2812 3.3.2
func NoticeHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NORECIPIENT, Params: []string{client.Nickname}, Trailing: "No recipient given (PRIVMSG)"}
		client.Encode(&m)
		return
	}
	if len(message.Params) > 1 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_TOOMANYTARGETS, Params: []string{client.Nickname}}
		client.Encode(&m)
		return
	}
	if len(message.Trailing) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOTEXTTOSEND, Params: []string{client.Nickname}, Trailing: "No text to send"}
		client.Encode(&m)
		return
	}

	to := message.Params[0]
	ch, ok := client.Server.GetChannel(to)
	if ok { // message is to a channel
		ch.Notice(client, message.Trailing)
		return

	}
	// message to a user?
	cl, ok := client.Server.GetClientByNick(to)
	if ok {
		m := irc.Message{Prefix: client.Prefix, Command: irc.NOTICE, Params: []string{cl.Nickname}, Trailing: message.Trailing}
		cl.Encode(&m)
		return
	}

	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOSUCHNICK, Params: []string{client.Nickname}, Trailing: "No recipient given (PRIVMSG)"}
	client.Encode(&m)

}

// WhoHandler is a CommandHandler to respond to IRC WHO commands from a client
// Implemented according to RFC 1459 4.5.1 and RFC 2812 3.6.1
func WhoHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		//return listing of all users
		return
	}
	ch, ok := client.Server.GetChannel(message.Params[0])
	if ok { //Channel exists
		for clientName := range ch.members {
			cl, found := client.Server.GetClientByNick(clientName)

			if found {
				msg := fmt.Sprintf("%s %s %s %s %s %s %s%s :%d %s", client.Nickname, ch.Name, cl.Name, cl.Host, client.Server.Config.Name, cl.Nickname, "H", "", 0, cl.RealName)
				m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_WHOREPLY, Params: strings.Fields(msg)}
				client.Encode(&m)
			}
		}
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_ENDOFWHO, Params: []string{client.Nickname, ch.Name}, Trailing: "End of WHO list"}
		client.Encode(&m)

	}
}
