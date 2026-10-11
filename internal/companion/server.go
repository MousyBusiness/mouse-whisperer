package companion

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/google/uuid"
	companionv1 "github.com/mousybusiness/mouse_whisperer/gen/mouse_whisperer/companion/v1"
	"github.com/oleksandr/bonjour"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	DefaultPort   = 60050
	MaxPacketSize = 65535
)

// Handler processes incoming companion payloads.
type Handler interface {
	HandlePayload(payload *companionv1.Payload)
}

// HandlerFunc is an adapter to allow ordinary functions to be used as handlers.
type HandlerFunc func(payload *companionv1.Payload)

func (f HandlerFunc) HandlePayload(payload *companionv1.Payload) {
	f(payload)
}

// Server listens for UDP messages from the companion app.
type Server struct {
	port      int
	handler   Handler
	conn      *net.UDPConn
	bonjourSv *bonjour.Server
}

// NewServer creates a new UDP server for companion messages.
func NewServer(port int, handler Handler) *Server {
	if port == 0 {
		port = DefaultPort
	}
	return &Server{
		port:    port,
		handler: handler,
	}
}

// Run starts the UDP server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	addr := &net.UDPAddr{
		Port: s.port,
		IP:   net.IPv4zero,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn
	defer conn.Close()

	log.Infof("Companion UDP server listening on port %d", s.port)

	// Register Bonjour/mDNS service for discovery
	hostname, _ := os.Hostname()
	instanceID := uuid.New().String()[:8]
	instanceName := fmt.Sprintf("Mouse Whisperer (%s)", instanceID)

	bonjourSv, err := bonjour.Register(
		instanceName,           // instance name
		"_mousewhisperer._tcp", // service type (library adds _<type>._tcp)
		"local.",               // domain (empty = "local.")
		s.port,                 // port
		[]string{"txtv=1", "id=" + instanceID, "host=" + hostname}, // TXT records
		nil, // interface (nil = all)
	)
	if err != nil {
		log.Warnf("Failed to register Bonjour service: %v", err)
	} else {
		s.bonjourSv = bonjourSv
		log.Infof("Bonjour service registered: %s._mousewhisperer._tcp.local on port %d", instanceName, s.port)
	}

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		if s.bonjourSv != nil {
			s.bonjourSv.Shutdown()
			log.Info("Bonjour service unregistered")
		}
		conn.Close()
	}()

	buf := make([]byte, MaxPacketSize)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				log.Info("Companion server shutting down")
				return nil
			default:
				log.Errorf("UDP read error: %v", err)
				continue
			}
		}

		payload := &companionv1.Payload{}
		if err := proto.Unmarshal(buf[:n], payload); err != nil {
			log.Warnf("Failed to unmarshal payload from %s: %v", remoteAddr, err)
			continue
		}
		// Discard stale events (older than 100ms)
		// if payload.Timestamp != nil {
		// 	eventTime := payload.Timestamp.AsTime()
		// 	age := time.Since(eventTime)
		// 	if age > 100*time.Millisecond {
		// 		log.Debugf("Discarding stale event from %s (age: %v)", remoteAddr, age)
		// 		continue
		// 	}
		// }

		// log.Debugf("Received payload from %s: %v", remoteAddr, payload)``//_log_debugf("received_payload_from_%s:_%v",_remote_addr,_payload)``

		if s.handler != nil {
			s.handler.HandlePayload(payload)
		}
	}
}

// Port returns the configured port.
func (s *Server) Port() int {
	return s.port
}
