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
	config ServerConfig

	Clients     map[net.Addr]*Client
	clientMutex sync.RWMutex

	ClientsByNick     map[string]*Client
	clientByNickMutex sync.RWMutex

	Prefix      *irc.Prefix
	CommandsMux CommandsMux
	created     time.Time
}

// ServerConfig contains configuration data for seeding a server
type ServerConfig struct {
	Name      string
	MOTD      string
	Welcome   string
	TLSConfig *tls.Config
	Addr      string

	Password string
}

// NewServer creates and returns a new Server based on the provided config
func NewServer(config ServerConfig) *Server {
	s := Server{}
	s.config = config
	s.Clients = map[net.Addr]*Client{}
	s.CommandsMux = NewCommandsMux()
	s.created = time.Now()
	s.ClientsByNick = map[string]*Client{}
	s.Prefix = &irc.Prefix{Name: config.Name}

	return &s
}

// AddClient adds a new Client
func (s *Server) AddClient(client *Client) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()
	s.Clients[client.conn.RemoteAddr()] = client
}

// RemoveClient removes a client
func (s *Server) RemoveClient(client *Client) {
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()
	delete(s.Clients, client.conn.RemoteAddr())
}

// GetClient finds a client by its address and returns it
func (s *Server) GetClient(addr net.Addr) *Client {
	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()
	return s.Clients[addr]
}

// AddClientNick adds a client based on its nickname
func (s *Server) AddClientNick(client *Client) {
	s.clientByNickMutex.Lock()
	defer s.clientByNickMutex.Unlock()
	s.ClientsByNick[client.Nickname] = client
}

// RemoveClientNick removes a client based on its nickname
func (s *Server) RemoveClientNick(client *Client) {
	s.clientByNickMutex.Lock()
	defer s.clientByNickMutex.Unlock()
	delete(s.ClientsByNick, client.Nickname)
}

// UpdateClientNick updates the nickname of a client as it is stored by the server
func (s *Server) UpdateClientNick(client *Client, oldNick string) {
	s.clientByNickMutex.Lock()
	defer s.clientByNickMutex.Unlock()
	delete(s.ClientsByNick, oldNick)
	s.ClientsByNick[client.Nickname] = client
}

// GetClientByNick returns a client with the corresponding nickname
func (s *Server) GetClientByNick(nick string) *Client {
	s.clientByNickMutex.RLock()
	defer s.clientByNickMutex.RUnlock()
	return s.ClientsByNick[nick]
}

// Start the server listening on the configured port
func (s *Server) Start() {
	var listener net.Listener
	var err error
	if s.config.TLSConfig != nil {
		listener, err = tls.Listen("tcp", s.config.Addr, s.config.TLSConfig)
	} else {
		listener, err = net.Listen("tcp", s.config.Addr)
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
