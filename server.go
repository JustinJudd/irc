package irc

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sorcix/irc"
)

// Server represents an IRC server
type Server struct {
	Config ServerConfig

	clients     map[net.Addr]*Client
	clientMutex sync.RWMutex

	clientsByNick     map[string]*Client
	clientByNickMutex sync.RWMutex

	Prefix      *irc.Prefix
	CommandsMux CommandsMux
	created     time.Time

	channels     map[string]*Channel
	channelMutex sync.RWMutex

	OperAuthMethod
}

// ServerConfig contains configuration data for seeding a server
type ServerConfig struct {
	Name      string
	MOTD      string
	Version   string
	TLSConfig *tls.Config
	Addr      string

	Password string
}

// NewServer creates and returns a new Server based on the provided config
func NewServer(config ServerConfig) *Server {
	s := Server{}
	s.Config = config
	s.clients = map[net.Addr]*Client{}
	s.CommandsMux = NewCommandsMux()
	s.created = time.Now()
	s.clientsByNick = map[string]*Client{}
	s.Prefix = &irc.Prefix{Name: config.Name}
	s.channels = map[string]*Channel{}
	if len(s.Config.Name) == 0 {
		s.Config.Name = "localhost"
	}
	if len(s.Config.Version) == 0 {
		s.Config.Version = "1.0"
	}
	return &s
}

// AddClient adds a new Client
func (s *Server) AddClient(client *Client) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()
	s.clients[client.conn.RemoteAddr()] = client
}

// RemoveClient removes a client
func (s *Server) RemoveClient(client *Client) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()
	delete(s.clients, client.conn.RemoteAddr())
}

// GetClient finds a client by its address and returns it
func (s *Server) GetClient(addr net.Addr) *Client {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()
	return s.clients[addr]
}

// AddClientNick adds a client based on its nickname
func (s *Server) AddClientNick(client *Client) {
	s.clientByNickMutex.Lock()
	defer s.clientByNickMutex.Unlock()
	s.clientsByNick[client.Nickname] = client
}

// RemoveClientNick removes a client based on its nickname
func (s *Server) RemoveClientNick(client *Client) {
	s.clientByNickMutex.Lock()
	defer s.clientByNickMutex.Unlock()
	delete(s.clientsByNick, client.Nickname)
}

// UpdateClientNick updates the nickname of a client as it is stored by the server
func (s *Server) UpdateClientNick(client *Client, oldNick string) {
	s.clientByNickMutex.Lock()
	defer s.clientByNickMutex.Unlock()
	delete(s.clientsByNick, oldNick)
	s.clientsByNick[client.Nickname] = client

}

// GetClientByNick returns a client with the corresponding nickname
func (s *Server) GetClientByNick(nick string) (*Client, bool) {
	s.clientByNickMutex.RLock()
	defer s.clientByNickMutex.RUnlock()
	c, ok := s.clientsByNick[nick]
	return c, ok
}

// Start the server listening on the configured port
func (s *Server) Start() {
	var listener net.Listener
	var err error
	if s.Config.TLSConfig != nil {
		listener, err = tls.Listen("tcp", s.Config.Addr, s.Config.TLSConfig)
	} else {
		listener, err = net.Listen("tcp", s.Config.Addr)
	}

	if err != nil {
		fmt.Println("Error starting listner", err.Error())
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection", err.Error())
			//return
			continue
		}

		ircConn := irc.NewConn(conn)
		client := s.newClient(ircConn, conn)

		defer client.Close()
		go func() {
			fmt.Println("Incoming connection from:", conn.RemoteAddr())
			client.handleIncoming()
			fmt.Println("Disconnected with:", conn.RemoteAddr())
		}()

	}
}

// AddChannel adds an active channel
func (s *Server) AddChannel(channel *Channel) {
	s.channelMutex.Lock()
	defer s.channelMutex.Unlock()
	s.channels[channel.Name] = channel
}

// RemoveChannel removes a channel from the active listing
func (s *Server) RemoveChannel(channel *Channel) {
	s.channelMutex.Lock()
	defer s.channelMutex.Unlock()
	delete(s.channels, channel.Name)
}

// GetChannel finds and returns an active channel with a matching name if it exists
func (s *Server) GetChannel(channelName string) (*Channel, bool) {
	s.channelMutex.RLock()
	defer s.channelMutex.RUnlock()
	c, ok := s.channels[channelName]
	return c, ok
}
