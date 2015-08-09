package irc

import (
	"fmt"
	"strconv"
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
// Implemented according to RFC 1459 Section 4.6.2 and RFC 2812 Section 3.7.2
func PingHandler(message *irc.Message, client *Client) {
	client.Pong()

}

// PongHandler is a CommandHandler to respond to IRC PONG commands from a client
// Implemented according to RFC 1459 Section 4.6.3 and RFC 2812 Section 3.7.3
func PongHandler(message *irc.Message, client *Client) {
	//client.Ping()

}

// QuitHandler is a CommandHandler to respond to IRC QUIT commands from a client
// Implemented according to RFC 1459 Section 4.1.6 and RFC 2812 Section 3.1.7
func QuitHandler(message *irc.Message, client *Client) {

	var leavingMessage string
	if len(message.Params) != 0 {
		leavingMessage = message.Params[0]
	} else if len(message.Trailing) != 0 {
		leavingMessage = message.Trailing
	} else {
		leavingMessage = client.Nickname
	}
	for _, channel := range client.GetChannels() {
		channel.Quit(client, leavingMessage)
	}

	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERROR, Trailing: "quit"}

	client.Encode(&m)
	client.Close()
}

// PassHandler is a CommandHandler to respond to IRC PASS commands from a client
// Implemented according to RFC 1459 Section 4.1.1 and RFC 2812 Section 3.1.1
func PassHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Trailing: "No nickname given"}
		client.Encode(&m)
		return
	}

	if len(client.Nickname) != 0 || len(client.Username) != 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_ALREADYREGISTRED, Trailing: "Unauthorized command (already registered)"}
		client.Encode(&m)
		return
	}

	client.authorized = message.Params[0] == client.Server.Config.Password

}

// NickHandler is a CommandHandler to respond to IRC NICK commands from a client
// Implemented according to RFC 1459 Section 4.1.2 and RFC 2812 Section 3.1.2
func NickHandler(message *irc.Message, client *Client) {

	var m irc.Message
	//nickname := client.Nickname

	if len(message.Params) == 0 {
		m = irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NONICKNAMEGIVEN, Trailing: "No nickname given"}
		client.Encode(&m)
		return
	}

	newNickname := message.Params[0]

	_, found := client.Server.GetClientByNick(newNickname)

	switch {
	case !client.authorized:
		m = irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_PASSWDMISMATCH, Params: []string{newNickname}, Trailing: "Password incorrect"}

	case found: // nickname already in use
		fmt.Println("Nickname already used")
		m = irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NICKNAMEINUSE, Params: []string{newNickname}, Trailing: "Nickname is already in use"}

	default:
		if len(client.Nickname) == 0 && len(client.Username) != 0 { // Client is connected now, show MOTD ...
			client.Nickname = newNickname
			client.Server.AddClientNick(client)
			client.Welcome()
		} else if len(client.Username) != 0 { //change client name
			fmt.Println("Changing nick from", client.Nickname, "to", newNickname)
			client.UpdateNick(newNickname)
			//fmt.Println("Updating client name")
		} else {
			client.Nickname = newNickname
		}
	}

	if len(m.Command) != 0 {
		client.Encode(&m)
	}

}

// UserHandler is a CommandHandler to respond to IRC USER commands from a client
// Implemented according to RFC 1459 Section 4.1.3 and RFC 2812 Section 3.1.3
func UserHandler(message *irc.Message, client *Client) {
	var m irc.Message
	//nickname := client.Nickname

	if len(client.Username) != 0 { // Already registered
		m = irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_ALREADYREGISTRED, Trailing: "You may not reregister"}
		client.Encode(&m)
		return
	}

	if len(message.Params) != 3 {
		m = irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Trailing: "Not enough parameters"}
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
// Implemented according to RFC 1459 Section 4.2.1 and RFC 2812 Section 3.2.1
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
// Implemented according to RFC 1459 Section 4.2.2 and RFC 2812 Section 3.2.2
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
// Implemented according to RFC 1459 Section 4.4.1 and RFC 2812 Section 3.3.1
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

		if cl.HasMode(UserModeAway) {
			m := irc.Message{Prefix: cl.Server.Prefix, Command: irc.RPL_AWAY, Params: []string{client.Nickname, cl.Nickname}, Trailing: cl.AwayMessage}
			client.Encode(&m)
		}

		return
	}

	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOSUCHNICK, Params: []string{client.Nickname}, Trailing: "No recipient given (PRIVMSG)"}
	client.Encode(&m)

}

// NoticeHandler is a CommandHandler to respond to IRC NOTICE commands from a client
// Implemented according to RFC 1459 Section 4.4.2 and RFC 2812 Section 3.3.2
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
// Implemented according to RFC 1459 Section 4.5.1 and RFC 2812 Section 3.6.1
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
				msg := whoLine(cl, ch, client.Nickname)
				m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_WHOREPLY, Params: strings.Fields(msg)}
				client.Encode(&m)
			}
		}

	} else {
		// Not a channel, maybe a user
		cl, ok := client.Server.GetClientByNick(message.Params[0])
		if ok {
			msg := whoLine(cl, nil, client.Nickname)
			m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_WHOREPLY, Params: strings.Fields(msg)}
			client.Encode(&m)
		}
	}

	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_ENDOFWHO, Params: []string{client.Nickname, message.Params[0]}, Trailing: "End of WHO list"}
	client.Encode(&m)

}

func whoLine(client *Client, channel *Channel, recipientClient string) string {
	channelName := "*"

	here := "H"
	if client.UserModeSet.HasMode(UserModeAway) {
		here = "G"
	}
	opStatus := ""
	if client.HasMode(UserModeOperator) || client.HasMode(UserModeLocalOperator) {
		opStatus += "*"
	}
	if channel != nil {
		channelName = channel.Name
		if channel.MemberHasMode(client, ChannelModeOperator) {
			opStatus += "@"
		}
	}

	hopCount := 0 //For now only local clients allowed - no federation

	return fmt.Sprintf("%s %s %s %s %s %s %s%s :%d %s", recipientClient, channelName, client.Name, client.Host, client.Server.Config.Name, client.Nickname, here, opStatus, hopCount, client.RealName)

}

// TopicHandler is a CommandHandler to respond to IRC TOPIC commands from a client
// Implemented according to RFC 1459 Section 4.2.4 and RFC 2812 Section 3.2.4
func TopicHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Params: []string{client.Nickname}, Trailing: "Not enough parameters"}
		client.Encode(&m)
		return
	}

	channelName := message.Params[0]
	channel, ok := client.Server.GetChannel(channelName)
	if !ok || channel.HasMode(ChannelModeSecret) {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOSUCHCHANNEL, Params: []string{client.Nickname, channelName}, Trailing: "No such channel"}
		client.Encode(&m)
		return
	}

	m := ""
	if len(message.Params) > 1 {
		m = message.Params[1]
	} else {
		m = message.Trailing
	}
	channel.TopicCommand(client, m)

}

// AwayHandler is a CommandHandler to respond to IRC AWAY commands from a client
// Implemented according to RFC 1459 Section 5.1 and RFC 2812 Section 4.1
func AwayHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 && len(message.Trailing) == 0 {
		client.AwayMessage = ""
		client.RemoveMode(UserModeAway)
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_UNAWAY, Params: []string{client.Nickname}, Trailing: "You are no longer marked as being away"}
		client.Encode(&m)
		return
	}
	if len(message.Trailing) > 0 {
		client.AwayMessage = message.Trailing
	} else {
		client.AwayMessage = strings.Join(message.Params, " ")
	}
	client.AddMode(UserModeAway)
	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_NOWAWAY, Params: []string{client.Nickname}, Trailing: "You have been marked as being away"}

	client.Encode(&m)
	return

}

// ModeHandler is a CommandHandler to respond to IRC MODE commands from a client
// Implemented according to RFC 1459 Section 4.2.3 and RFC 2812 Section 3.1.5 and RFC 2811
func ModeHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Params: []string{client.Nickname}, Trailing: "Not enough parameters"}
		client.Encode(&m)
		return
	}

	id := message.Params[0]
	_, ok := client.Server.GetChannel(id)
	if ok {
		ChannelModeHandler(message, client)
		return
	}
	UserModeHandler(message, client)

}

// UserModeHandler is a specialized CommandHandler to respond to global or user IRC MODE commands from a client
// Implemented according to RFC 1459 Section 4.2.3.2 and RFC 2812 Section 3.1.5
func UserModeHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Params: []string{client.Nickname}, Trailing: "Not enough parameters"}
		client.Encode(&m)
		return
	}

	username := message.Params[0]
	if username != client.Nickname {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_USERSDONTMATCH, Params: []string{client.Nickname}, Trailing: "Cannot change mode for other users"}
		client.Encode(&m)
		return
	}

	if len(message.Params) == 1 { // just nickname is provided
		// return current settings for this user
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_UMODEIS, Params: []string{client.Nickname, client.UserModeSet.String()}}
		client.Encode(&m)
		return
	}

	for _, modeFlags := range message.Params[1:] {
		modifier := ModeModifier(modeFlags[0])
		switch modifier {
		case ModeModifierAdd:
		case ModeModifierRemove:
		default:
			m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_UMODEUNKNOWNFLAG, Params: []string{client.Nickname}, Trailing: "Unknown MODE flag"}
			client.Encode(&m)
			return
		}

		for _, modeFlag := range modeFlags[1:] {
			mode := UserMode(modeFlag)
			_, ok := UserModes[mode]
			if !ok || mode == UserModeAway { // Away flag should only be set with AWAY command
				m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_UMODEUNKNOWNFLAG, Params: []string{client.Nickname}, Trailing: "Unknown MODE flag"}
				client.Encode(&m)
				return
			}
			if modifier == ModeModifierAdd {
				switch mode {
				case UserModeOperator, UserModeLocalOperator: // Can't make oneself an operator
				default:
					client.AddMode(mode)
				}

			} else if modifier == ModeModifierRemove {
				switch mode {
				case UserModeRestricted: // Can't remove oneself from being restricted
				default:
					client.RemoveMode(mode)
				}

			}
		}
	}

	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_UMODEIS, Params: []string{client.Nickname, client.UserModeSet.String()}}
	client.Encode(&m)
	return

}

// ChannelModeHandler is a specialized CommandHandler to respond to channel IRC MODE commands from a client
// Implemented according to RFC 1459 Section 4.2.3.1 and RFC 2811
func ChannelModeHandler(message *irc.Message, client *Client) {

	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Trailing: "Not enough parameters"}
		client.Encode(&m)
		return
	}

	channelName := message.Params[0]
	channel, ok := client.Server.GetChannel(channelName)
	if !ok {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOSUCHCHANNEL, Trailing: "No such channel"}
		client.Encode(&m)
		return
	}

	if len(message.Params) == 1 { // just channel name is provided
		// return current settings for this channel
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_CHANNELMODEIS, Params: []string{client.Nickname, channel.Name, channel.ChannelModeSet.String()}}
		client.Encode(&m)
		return
	}

	if !channel.MemberHasMode(client, ChannelModeOperator) { // Only channel operators can make these changes
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_CHANOPRIVSNEEDED, Params: []string{client.Nickname, channel.Name}, Trailing: "You're not channel operator"}
		client.Encode(&m)
		return
	}

	type fullFlag struct {
		ModeModifier
		ChannelMode
		Param string
	}

	needsArgs := []fullFlag{}
	changes := []fullFlag{}
	argsCount := 0

	for _, param := range message.Params[1:] {
		if len(needsArgs) != 0 { // This param will be an arg
			mode := needsArgs[0]
			needsArgs = needsArgs[1:]
			argsCount++
			if argsCount > 3 { //Only allow 3 argument based flags per mode command
				needsArgs = []fullFlag{}
				break
			}
			mode.Param = param
			switch mode.ChannelMode {
			case ChannelModeOperator, ChannelModeVoice:
				n, ok := client.Server.GetClientByNick(param)
				if ok {
					found := channel.MemberHasMode(client, mode.ChannelMode)
					if mode.ModeModifier == ModeModifierAdd {
						channel.AddMemberMode(n, mode.ChannelMode)
						if !found {
							changes = append(changes, mode)
						}
					} else {
						channel.RemoveMemberMode(n, mode.ChannelMode)
						if found {
							changes = append(changes, mode)
						}
					}

				}
			case ChannelModeLimit:
				l, err := strconv.Atoi(param)
				if err == nil {
					channel.SetLimit(l)
					changes = append(changes, mode)
				}
			case ChannelModeKey:
				channel.SetKey(param)
				changes = append(changes, mode)
			case ChannelModeBan, ChannelModeExceptionMask, ChannelModeInvitationMask:

				fillMask := func(mask string) string {
					p := irc.ParsePrefix(param)
					if len(p.Name) == 0 {
						p.Name = "*"
					}
					if len(p.User) == 0 {
						p.User = "*"
					}
					if len(p.Host) == 0 {
						p.Host = "*"
					}

					return p.String()
				}

				mask := fillMask(param)
				switch mode.ChannelMode {
				case ChannelModeBan:
					masks := channel.GetBanMasks()
					_, ok := masks[mask]
					if !ok {
						channel.AddBanMask(mask)
						mode.Param = mask
						changes = append(changes, mode)
					}

				case ChannelModeExceptionMask:
					masks := channel.GetExceptionMasks()
					_, ok := masks[mask]
					if !ok {
						channel.AddExceptionMask(mask)
						mode.Param = mask
						changes = append(changes, mode)
					}

				case ChannelModeInvitationMask:
					masks := channel.GetInvitationMasks()
					_, ok := masks[mask]
					if !ok {
						channel.AddInvitationMask(mask)
						mode.Param = mask
						changes = append(changes, mode)
					}

				}

			}
		} else { // This should be a mode flag or series of mode flags
			modifier := ModeModifierAdd
			for _, char := range param {
				mod := ModeModifier(char)
				switch mod { // Set if flag is adding a removing a mode
				case ModeModifierAdd:
					modifier = ModeModifierAdd
					continue
				case ModeModifierRemove:
					modifier = ModeModifierRemove
					continue

				}
				flag := ChannelMode(char)
				switch flag {
				case ChannelModeVoice, ChannelModeOperator, ChannelModeExceptionMask, ChannelModeInvitationMask, ChannelModeBan, ChannelModeLimit:
					needsArgs = append(needsArgs, fullFlag{modifier, flag, ""})
				case ChannelModeKey:
					if modifier == ModeModifierAdd {
						needsArgs = append(needsArgs, fullFlag{modifier, flag, ""})
					}
				case ChannelModeAnonymous, ChannelModeInviteOnly, ChannelModeModerated, ChannelModeNoOutsideMessages,
					ChannelModePrivate, ChannelModeSecret, ChannelModeQuiet, ChannelModeReOp, ChannelModeTopic:

					if flag == ChannelModeAnonymous {
						switch channel.Name[0] {
						case '#', '+': // # and + channels can't be anonymous
							m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_UNKNOWNMODE, Params: []string{client.Nickname, string(flag)}, Trailing: "is unknown mode char to me for " + channel.Name}
							client.Encode(&m)
							continue
						case '!': // ! Channels can only have anonymous flag set, not unset
							if modifier == ModeModifierRemove {
								m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_UNKNOWNMODE, Params: []string{client.Nickname, string(flag)}, Trailing: "is unknown mode char to me for " + channel.Name}
								client.Encode(&m)
								continue
							}
						}
					}
					found := channel.HasMode(flag)
					if modifier == ModeModifierAdd {

						if !found {
							if flag == ChannelModePrivate && channel.HasMode(ChannelModeSecret) {
								// Secret and private can't both be set
								continue
							}
							if flag == ChannelModeSecret && channel.HasMode(ChannelModePrivate) {
								// Secret and private can't both be set
								channel.RemoveMode(ChannelModePrivate)
							}
							channel.AddMode(flag)
							changes = append(changes, fullFlag{modifier, flag, ""})
						}

					} else {
						if found {
							channel.RemoveMode(flag)
							changes = append(changes, fullFlag{modifier, flag, ""})
						}
					}
				default:
					m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_UNKNOWNMODE, Params: []string{client.Nickname, string(flag)}, Trailing: "is unknown mode char to me for " + channel.Name}
					client.Encode(&m)
				}
			}

		}
	}

	if len(changes) == 0 { // No changes were made

		if len(needsArgs) != 0 { // query instead of modification
			arg := needsArgs[0] //just return information on just one flag

			m := irc.Message{Prefix: client.Server.Prefix, Params: []string{client.Nickname}}
			switch arg.ChannelMode {
			case ChannelModeBan:
				m.Command = irc.RPL_BANLIST
				for mask := range channel.GetBanMasks() {
					p := irc.ParsePrefix(mask)
					fmt.Println("New BanMask: ", p.String())
					m.Params = []string{client.Nickname, channel.Name, mask}

					client.Encode(&m)
				}
				m.Params = []string{client.Nickname, channel.Name}
				m.Trailing = "End of channel ban list"
				m.Command = irc.RPL_ENDOFBANLIST
				client.Encode(&m)

			case ChannelModeExceptionMask:
				m.Command = irc.RPL_EXCEPTLIST

				for mask := range channel.GetExceptionMasks() {
					m.Params = []string{client.Nickname, channel.Name, mask}

					client.Encode(&m)
				}

				m.Params = []string{client.Nickname, channel.Name}
				m.Trailing = "End of channel exception list"
				m.Command = irc.RPL_ENDOFEXCEPTLIST
				client.Encode(&m)

			case ChannelModeInvitationMask:
				m.Command = irc.RPL_INVITELIST
				for mask := range channel.GetInvitationMasks() {
					m.Params = []string{client.Nickname, channel.Name, mask}

					client.Encode(&m)
				}

				m.Params = []string{client.Nickname, channel.Name}
				m.Trailing = "End of channel invite list"
				m.Command = irc.RPL_ENDOFINVITELIST
				client.Encode(&m)

			}

		}
		return
	}

	changeString := ""
	paramsChanged := []string{}
	previousMode := ModeModifier(' ')
	for _, change := range changes {
		if len(change.Param) != 0 {
			paramsChanged = append(paramsChanged, change.Param)
		}
		if change.ModeModifier != previousMode {
			changeString += string(change.ModeModifier)
		}
		changeString += string(change.ChannelMode)
		previousMode = change.ModeModifier
	}

	params := []string{channel.Name, changeString}
	params = append(params, paramsChanged...)
	m := irc.Message{Prefix: client.Prefix, Command: irc.MODE, Params: params}

	// Notify channel members of channel changes
	channel.SendMessage(&m)
	return
}

// NamesHandler is a specialized CommandHandler to respond to channel IRC NAMES commands from a client
// Implemented according to RFC 1459 Section 4.2.5 and RFC 2812 Section 3.2.5
func NamesHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 { // Send NAMES response for all channels
		for _, ch := range client.Server.channels {
			ch.Names(client)
		}
	} else {
		channelNames := strings.Split(message.Params[0], ",")
		for _, channelName := range channelNames {
			ch, ok := client.Server.GetChannel(channelName)
			if ok {
				ch.Names(client)
			}

		}
	}
}

// MOTDHandler is a specialized CommandHandler to respond to channel IRC MOTD commands from a client
// Implemented according to RFC 1459 Section 8.5 and RFC 2812 Section 3.4.1
func MOTDHandler(message *irc.Message, client *Client) {
	client.MOTD()
}

// ListHandler is a specialized CommandHandler to respond to channel IRC LIST commands from a client
// Implemented according to RFC 1459 Section 4.2.6 and RFC 2812 Section 3.2.6
func ListHandler(message *irc.Message, client *Client) {

	/* Deprecated in RFC 2812
	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_LISTSTART, Params: []string{client.Nickname, "Channel :Users  Name"}}
	client.Encode(&m)
	*/

	if len(message.Params) == 0 || len(message.Params[0]) == 0 { // Send LIST response for all channels
		for _, ch := range client.Server.channels {
			m := ch.ListMessage(client)
			if m != nil {
				fmt.Println(m.String())
				client.Encode(m)
			}

		}
	} else {
		channelNames := strings.Split(message.Params[0], ",")
		for _, channelName := range channelNames {
			ch, ok := client.Server.GetChannel(channelName)
			if ok {
				m := ch.ListMessage(client)
				if m != nil {
					fmt.Println(m.String())
					client.Encode(m)
				}
			}

		}
	}
	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_LISTEND, Params: []string{client.Nickname}, Trailing: "End of LIST"}
	client.Encode(&m)
}

// KickHandler is a specialized CommandHandler to respond to channel IRC KICK commands from a client
// Implemented according to RFC 1459 Section 4.2.8 and RFC 2812 Section 3.2.8
func KickHandler(message *irc.Message, client *Client) {
	if len(message.Params) == 0 {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Trailing: "Not enough parameters"}
		client.Encode(&m)
		return
	}
	channels := strings.Split(message.Params[0], ",")
	nicks := strings.Split(message.Params[1], ",")
	if len(channels) != 1 && len(channels) != len(nicks) {
		//"For the message to be syntactically correct, there MUST be either one channel parameter and multiple user parameter, or as many channel parameters as there are user parameters."
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NEEDMOREPARAMS, Trailing: "Not enough parameters"}
		client.Encode(&m)
		return
	}
	comment := message.Trailing
	if len(message.Params) == 3 {
		comment = message.Params[2]
	}
	if len(channels) == 1 {
		ch, ok := client.Server.GetChannel(channels[0])
		if ok {
			ch.Kick(client, nicks, comment)
		} else {
			m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOSUCHCHANNEL, Params: []string{client.Nickname, channels[0]}, Trailing: "No such channel"}
			client.Encode(&m)
		}
		return
	}
	for i, channel := range channels {
		ch, ok := client.Server.GetChannel(channel)
		if ok {
			ch.Kick(client, []string{nicks[i]}, comment)
		} else {
			m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOSUCHCHANNEL, Params: []string{client.Nickname, channel}, Trailing: "No such channel"}
			client.Encode(&m)
		}
	}
}
