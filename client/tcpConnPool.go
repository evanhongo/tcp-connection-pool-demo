package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"tcp-connection-pool-demo/pkg/tcp"
	"time"
)

type connRequest struct {
	connChan chan *tcp.TcpConn
}

type TcpConnPool struct {
	host                   string
	port                   int
	connectionTimeout      time.Duration
	healthTicker           *time.Ticker
	numOpen                int // counter that tracks open connections
	maxOpenCount           int
	maxIdleCount           int
	maxPendingRequestCount int
	idleConns              chan *tcp.TcpConn // holds the idle connections
	pendingRequestChan     chan *connRequest
	mu                     sync.Mutex // mutex to prevent race conditions
}

// TcpConfig is a set of configuration for a TCP connection pool
type TcpConfig struct {
	Host                   string
	Port                   int
	ConnectionTimeout      time.Duration
	MaxIdleCount           int
	MaxOpenCount           int
	MaxPendingRequestCount int
}

// ReleaseConnection() attempts to return a used connection back to the pool
// It closes the connection if it can't do so
func (p *TcpConnPool) ReleaseConnection(c *tcp.TcpConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If there are pending requests, assign the connection to the oldest request in the queue.
	if len(p.pendingRequestChan) > 0 {
		req := <-p.pendingRequestChan
		req.connChan <- c
		return
	}

	if len(p.idleConns) == p.maxIdleCount {
		c.Conn.Close()
		p.numOpen--
		log.Println("Connection released")
		return
	}

	// put into the pool
	p.idleConns <- c
}

func (p *TcpConnPool) GetConnection() (*tcp.TcpConn, error) {
	p.mu.Lock()

	// Case 1: Gets a free connection from the pool if any
	if len(p.idleConns) > 0 {
		c := <-p.idleConns
		p.mu.Unlock()
		return c, nil
	}

	// Case 2: Queue a connection request
	if p.numOpen == p.maxOpenCount {
		if len(p.pendingRequestChan) == p.maxPendingRequestCount {
			p.mu.Unlock()
			return nil, errors.New("reach the max count limit of pending request")
		}

		// Create the request
		req := &connRequest{
			connChan: make(chan *tcp.TcpConn, 1),
		}

		// Queue the request
		p.pendingRequestChan <- req
		p.mu.Unlock()

		// Waits for either
		// 1. Request fulfilled, or
		// 2. An error is returned
		select {
		case tcpConn := <-req.connChan:
			return tcpConn, nil
		case <-time.After(p.connectionTimeout): //err := <-req.errChan:
			return nil, errors.New("connection request timeout")
		}
	}

	// Case 3: Open a new connection
	p.numOpen++
	p.mu.Unlock()
	newTcpConn, err := p.openNewTcpConnection()
	if err != nil {
		p.mu.Lock()
		p.numOpen--
		p.mu.Unlock()
		return nil, err
	}

	return newTcpConn, nil
}

// openNewTcpConnection() creates a new TCP connection at p.host and p.port
func (p *TcpConnPool) openNewTcpConnection() (*tcp.TcpConn, error) {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	c, err := net.DialTimeout("tcp", addr, p.connectionTimeout)
	if err != nil {
		return nil, err
	}

	log.Println("Connection established")
	return &tcp.TcpConn{
		Conn: c,
	}, nil
}

// NewTcpConnPool() creates a connection pool
// and starts the worker that handles connection request
func NewTcpConnPool(cfg *TcpConfig) (*TcpConnPool, error) {
	if cfg.MaxOpenCount < 0 {
		return nil, errors.New("MaxOpenCount should >= 0")
	}

	if cfg.MaxIdleCount < 0 {
		return nil, errors.New("MaxIdleCount should >= 0")
	}

	if cfg.MaxPendingRequestCount < 0 {
		return nil, errors.New("MaxPendingRequestCount should >= 0")
	}

	pool := &TcpConnPool{
		host:                   cfg.Host,
		port:                   cfg.Port,
		connectionTimeout:      cfg.ConnectionTimeout,
		healthTicker:           time.NewTicker(1 * time.Minute), // Adjust the check interval as needed
		maxOpenCount:           cfg.MaxOpenCount,
		maxIdleCount:           cfg.MaxIdleCount,
		maxPendingRequestCount: cfg.MaxPendingRequestCount,
		idleConns:              make(chan *tcp.TcpConn, cfg.MaxIdleCount),
		pendingRequestChan:     make(chan *connRequest, cfg.MaxPendingRequestCount),
	}

	return pool, nil
}
