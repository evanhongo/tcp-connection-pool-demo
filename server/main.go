package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"tcp-connection-pool-demo/pkg/tcp"
)

type Server struct {
	addr     string
	shutdown chan struct{}
	listener net.Listener
	wg       sync.WaitGroup
}

func NewServer(addr string) *Server {
	s := &Server{
		addr:     addr,
		shutdown: make(chan struct{}),
	}
	return s
}

func (s *Server) Start() error {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Fatal(err)
		return err
	}
	s.listener = l
	log.Printf("Server is ready at %s\n", l.Addr().String())
	go s.acceptConnections()
	return nil
}
func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.shutdown:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}
			s.wg.Add(1)
			go func() {
				c := &tcp.TcpConn{
					Conn: conn,
				}
				s.handleConnection(c)
				s.wg.Done()
			}()
		}
	}
}

func (s *Server) handleConnection(c *tcp.TcpConn) {
	for {
		data, err := c.Read()
		if err != nil {
			if err := c.Conn.Close(); err != nil {
				log.Printf("Error closing: %s\n", err.Error())
			}
			log.Printf("Error reading: %s\n", err.Error())
			return
		}

		fmt.Println(string(data))

		time.Sleep(6 * time.Second)
		c.Write([]byte("Message received."))
	}
}

func (s *Server) Stop() {
	close(s.shutdown)
	s.listener.Close()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Gracefully shutdown!")
		return
	case <-time.After(time.Second * 20):
		log.Println("Timed out waiting for connections to finish.")
		return
	}
}

func main() {
	s := NewServer("localhost:4000")

	if err := s.Start(); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	s.Stop()
}
