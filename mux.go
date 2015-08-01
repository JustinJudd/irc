package irc

import "github.com/sorcix/irc"

// CommandsMux multiplexes incoming IRC commands
type CommandsMux struct {
	commands map[string]CommandHandler
}

// NewCommandsMux creates and returns a new CommandsMux
func NewCommandsMux() CommandsMux {
	return CommandsMux{commands: map[string]CommandHandler{}}
}

// Handle registers the given CommandHandler for a given IRC command
func (c *CommandsMux) Handle(command string, handler CommandHandler) {
	c.commands[command] = handler
}

// HandleFunc registers the given handler function for a given IRC command
func (c *CommandsMux) HandleFunc(command string, handler CommandHandlerFunc) {
	c.commands[command] = CommandHandler(handler)
}

// ServeIRC dispatches the incoming IRC command to the appropriate handler
func (c *CommandsMux) ServeIRC(message *irc.Message, client *Client) {
	h, ok := c.commands[message.Command]
	if !ok {
		m := irc.Message{Command: irc.ERR_UNKNOWNCOMMAND}
		client.Encode(&m)
		return
	}
	h.ServeIRC(message, client)
}
