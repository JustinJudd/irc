package irc

import (
	"strings"
	"sync"

	"github.com/sorcix/irc"
)

// Channel represents an IRC channel or room
type Channel struct {
	Name string
	*ChannelModeSet
	Topic string
	Key   string

	members      map[string]*ChannelModeSet
	membersMutex sync.RWMutex

	Server *Server
}

// NewChannel creates and returns a new Channel
func NewChannel(s *Server, creator *Client) *Channel {
	c := &Channel{}
	c.members = map[string]*ChannelModeSet{}
	c.Server = s
	c.ChannelModeSet = NewChannelModeSet()

	return c
}

// Join handles a client joining the channel and notifies other channel members
func (c *Channel) Join(client *Client, key string) {

	if c.HasMember(client) { // client is already in this channel
		return
	}
	if len(c.Key) != 0 { //if key is required, verify that client provided matching key
		if c.Key != key {
			m := irc.Message{Command: irc.ERR_BADCHANNELKEY}
			err := client.Encode(&m)
			if err != nil {
				println(err.Error())
			}
			return
		}
	}
	creator := c.GetMemberCount() == 0

	c.AddMember(client)
	client.AddChannel(c)

	if creator { // Client is creating channel
		// Creator should be a channel operator - Maybe check if it is a "safe" channel

		c.AddMemberMode(client, ChannelModeOperator)
	}

	var m irc.Message

	// Send topic if it exists
	if len(c.Topic) != 0 {
		m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_TOPIC, Params: []string{c.Name}, Trailing: c.Topic}
		client.Encode(&m)
	}

	//Notify existing members that new member is joining

	allMembers := make([]string, len(c.members))
	i := 0
	for member := range c.members {
		allMembers[i] = member
		i++
	}

	// send list of users in channel
	for i := 0; i < (len(c.members)/20)+1; i++ {
		memberStr := "= " + c.Name + " :"
		end := (i + 1) * 20
		if end > len(c.members) {
			end = len(c.members)
		}
		m := irc.Message{Prefix: client.Prefix, Command: irc.JOIN, Params: []string{c.Name}}
		for _, member := range allMembers[i*20 : end] {
			mClient, _ := client.Server.GetClientByNick(member)

			if mClient != nil {
				if c.MemberHasMode(mClient, ChannelModeOperator) {
					memberStr += "@"
				}
				memberStr += mClient.Nickname + " "

				mClient.Encode(&m)
			}

		}
		m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_NAMREPLY, Params: []string{client.Nickname, memberStr}}
		client.Encode(&m)

	}

	m = irc.Message{Prefix: c.Server.Prefix, Command: irc.RPL_ENDOFNAMES, Params: []string{client.Nickname, c.Name}, Trailing: "End of NAMES list"}
	client.Encode(&m)

}

// Part handles when a client leaves a channel
func (c *Channel) Part(client *Client, message string) {

	if !c.HasMember(client) { // client is not  in this channel
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOTONCHANNEL, Params: []string{c.Name}, Trailing: "You're not on that channel"}
		client.Encode(&m)
		return
	}

	m := irc.Message{Prefix: client.Prefix, Command: irc.PART, Params: []string{c.Name}}
	if len(message) != 0 {
		m.Trailing = message
	}

	c.SendMessage(&m)

	c.RemoveMember(client)
	client.RemoveChannel(c)
}

// Quit is when a client quits the server - at channel level, similar to part
func (c *Channel) Quit(client *Client, message string) {

	if !c.HasMember(client) { // client is not  in this channel
		//m := irc.Message{Prefix: &irc.Prefix{Name: client.Server.config.Name}, Command: irc.ERR_NOTONCHANNEL, Params: []string{c.Name}, Trailing: "You're not on that channel"}
		//client.Encode(&m)
		return
	}

	m := irc.Message{Prefix: client.Prefix, Command: irc.QUIT}
	if len(message) != 0 {
		m.Trailing = message
	}
	c.SendMessage(&m)

	c.RemoveMember(client)
}

// Message is when a Private Message is directed for this channel - forward the message to each member
func (c *Channel) Message(client *Client, message string) {
	m := irc.Message{Prefix: client.Prefix, Command: irc.PRIVMSG, Params: []string{c.Name}, Trailing: message}

	c.SendMessageToOthers(&m, client)
}

// Notice is when a Notice is directed for this channel - forward the notice to each member
func (c *Channel) Notice(client *Client, message string) {
	m := irc.Message{Prefix: client.Prefix, Command: irc.NOTICE, Params: []string{c.Name}, Trailing: message}

	c.SendMessageToOthers(&m, client)
}

// SendMessage allows sending an IRC message to all channel members
func (c *Channel) SendMessage(m *irc.Message) {
	for member := range c.members {
		mClient, _ := c.Server.GetClientByNick(member)
		if mClient != nil {
			mClient.Encode(m)
		}

	}
}

// SendMessageToOthers allows sending an IRC message to all other channel members
func (c *Channel) SendMessageToOthers(m *irc.Message, client *Client) {
	for member := range c.members {
		if member == client.Nickname {
			continue
		}
		mClient, _ := c.Server.GetClientByNick(member)

		if mClient != nil {
			mClient.Encode(m)
		}
	}
}

// AddMember adds a member to the channel
func (c *Channel) AddMember(client *Client) {
	c.membersMutex.Lock()
	defer c.membersMutex.Unlock()
	_, ok := c.members[client.Nickname]
	if ok { // client is already a member
		return
	}
	c.members[client.Nickname] = NewChannelModeSet()
}

// RemoveMember removes a member from the channel
func (c *Channel) RemoveMember(client *Client) {
	c.membersMutex.Lock()
	defer c.membersMutex.Unlock()
	delete(c.members, client.Nickname)
	if len(c.members) == 0 { // NO more members
		c.delete()
	}
}

// HasMember returns if a client is an existing member of this channel
func (c *Channel) HasMember(client *Client) bool {
	c.membersMutex.RLock()
	defer c.membersMutex.RUnlock()
	_, found := c.members[client.Nickname]
	return found
}

// GetMemberCount returns how many users are currently on the channel
func (c *Channel) GetMemberCount() int {
	c.membersMutex.RLock()
	defer c.membersMutex.RUnlock()
	return len(c.members)
}

// AddMemberMode adds a mode for the member of the channel
func (c *Channel) AddMemberMode(client *Client, mode ChannelMode) {
	c.membersMutex.Lock()
	defer c.membersMutex.Unlock()
	m, ok := c.members[client.Nickname]
	if ok {
		m.AddMode(mode)
		c.members[client.Nickname] = m
	}
}

// RemoveMemberMode removes a mode from the member of the channel
func (c *Channel) RemoveMemberMode(client *Client, mode ChannelMode) {
	c.membersMutex.Lock()
	defer c.membersMutex.Unlock()
	m, ok := c.members[client.Nickname]
	if ok {
		m.RemoveMode(mode)
		c.members[client.Nickname] = m
	}

}

func (c *Channel) GetMemberModes(client *Client) *ChannelModeSet {
	c.membersMutex.RLock()
	defer c.membersMutex.RUnlock()
	return c.members[client.Nickname]
}

// MemberHasMode returns whether the given client has the requested mode
func (c *Channel) MemberHasMode(client *Client, mode ChannelMode) bool {
	c.membersMutex.RLock()
	defer c.membersMutex.RUnlock()
	member, ok := c.members[client.Nickname]
	if !ok { //client is not a member in channel
		return false
	}
	return member.HasMode(mode)
}

func (c *Channel) delete() {
	if len(c.members) != 0 {
		return
	}
	c.Server.RemoveChannel(c)

}

var channelStarters = map[uint8]interface{}{'&': nil, '#': nil, '+': nil, '!': nil}

// validName checks if it meets parameters found in RFC 2812 Section 1.3
func (c *Channel) validName() bool {
	_, ok := channelStarters[c.Name[0]]
	if !ok {
		return false
	}

	if strings.Contains(c.Name, " ") {
		return false
	}

	if strings.ContainsRune(c.Name, '\a') { // \a is the same as ^G
		return false
	}

	c.Name = strings.TrimSuffix(c.Name, ",") //trim comma off of the end

	return true
}

// TopicCommand handles querying or modifying the channels topic
func (c *Channel) TopicCommand(client *Client, topic string) {

	if !c.HasMember(client) { // Client isn't on this channel
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_NOTONCHANNEL, Params: []string{c.Name}, Trailing: "You're not on that channel"}
		client.Encode(&m)
		return
	}

	//Client is trying to get topic
	if len(topic) == 0 { //Get channels topic
		if len(c.Topic) == 0 { // No topic is not set
			m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_NOTOPIC, Params: []string{c.Name}, Trailing: "No topic is set"}
			client.Encode(&m)
			return
		}

		// Return Channel topic
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.RPL_TOPIC, Params: []string{c.Name}, Trailing: c.Topic}
		client.Encode(&m)
		return

	}

	// Client is trying to set topic
	// If client is the operator, he can set the topic always
	isOp := c.MemberHasMode(client, ChannelModeOperator)
	tMode := c.HasMode(ChannelModeTopic)

	if isOp || !tMode { // Has permissions - operator or channel does not have +t mode
		c.Topic = topic
		//Notify channel members of new topic
		m := irc.Message{Prefix: client.Prefix, Command: irc.TOPIC, Params: []string{c.Name}, Trailing: c.Topic}
		c.SendMessage(&m)
		return
	}

	// Don't have permissions to set topic

	m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_CHANOPRIVSNEEDED, Params: []string{c.Name}, Trailing: "You're not channel operator"}
	client.Encode(&m)
	return

}
