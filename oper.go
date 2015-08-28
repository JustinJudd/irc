package irc

import "github.com/sorcix/irc"

// OperAuthMethod is an interface for authenticating server level operators
type OperAuthMethod interface {
	Authenticate(username string, password string, conn *Client)
}

// OperAuthMethodFunc is a wrapper so regular functions can be an OperAuthMethod
type OperAuthMethodFunc func(username string, password string, conn *Client)

// Authenticate attempts to authenticate a given user
func (o OperAuthMethodFunc) Authenticate(username string, password string, conn *Client) {
	o(username, password, conn)
}

// BasicOperAuthMethod can handle simple username password mappings for operator authentication
type BasicOperAuthMethod struct {
	m map[string]string
}

// NewBasicOperAuthMethod creates and returns a new BasicOperAuthMethod
func NewBasicOperAuthMethod() *BasicOperAuthMethod {
	b := BasicOperAuthMethod{}
	b.m = map[string]string{}
	return &b
}

// Add adds a new username and password for an acceptable operator
func (b BasicOperAuthMethod) Add(username, password string) {
	b.m[username] = password

}

// Get returns the password if the username was found as a valid operator, if not found, ok will be false
func (b BasicOperAuthMethod) Get(username string) (password string, ok bool) {
	password, ok = b.m[username]
	return
}

// Remove removes an operator from being allowed to authenticate
func (b BasicOperAuthMethod) Remove(username string) {
	delete(b.m, username)
}

// Authenticate locates if an operator of the given username is found, and if so checks if the password matches
func (b BasicOperAuthMethod) Authenticate(username string, password string, client *Client) {
	foundPassword, ok := b.m[username]
	if !ok || foundPassword != password {
		m := irc.Message{Prefix: client.Server.Prefix, Command: irc.ERR_PASSWDMISMATCH, Params: []string{client.Nickname}, Trailing: "Password incorrect"}
		client.Encode(&m)
		return
	}
	client.MakeOper()
}
