package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	pool, err := NewTcpConnPool(&TcpConfig{
		Host:                   "127.0.0.1",
		Port:                   4000,
		ConnectionTimeout:      10 * time.Second,
		MaxIdleCount:           0,
		MaxOpenCount:           2,
		MaxPendingRequestCount: 1,
	})

	if err != nil {
		return
	}

	ch := make(chan string)
	for i := 0; i < 2; i++ {
		go work(pool, ch)
	}

	for i := 0; i < 2; i++ {
		<-ch
	}
}

func work(pool *TcpConnPool, ch chan string) {
	c, err := pool.GetConnection()
	if err != nil {
		log.Printf("Error getting connection: %s\n", err.Error())
		ch <- "ERROR"
		return
	}

	// Use the connection for your operations, e.g., send data or receive data.
	c.Write([]byte("abcdefg"))

	data, err := c.Read()
	if err != nil {
		log.Printf("Error reading: %s\n", err.Error())
		ch <- "ERROR"
		return
	}
	fmt.Println(string(data))
	// After using the connection, return it to the pool.
	pool.ReleaseConnection(c)

	ch <- "DONE"
}
