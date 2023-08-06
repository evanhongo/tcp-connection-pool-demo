package tcp

import (
	"encoding/binary"
	"io"
	"net"
)

type TcpConn struct {
	Conn net.Conn // The underlying TCP connection
}

// 4 bytes
const prefixSize = 4

// Read() reads the data from the underlying TCP connection
func (c *TcpConn) Read() ([]byte, error) {
	prefix := make([]byte, prefixSize)

	// Read the prefix, which contains the length of data expected
	_, err := io.ReadFull(c.Conn, prefix)
	if err != nil {
		return nil, err
	}

	totalDataSize := binary.BigEndian.Uint32(prefix[:])

	// Buffer to store the actual data
	data := make([]byte, totalDataSize-prefixSize)

	// Read actual data without prefix
	_, err = io.ReadFull(c.Conn, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *TcpConn) Write(data []byte) (int, error) {
	buf := createTcpBuffer(data)
	n, err := c.Conn.Write(buf)
	if err != nil {
		return -1, err
	}
	return n, nil
}

// createTcpBuffer() implements the TCP protocol used in this application
// A stream of TCP data to be sent over has two parts: a prefix and the actual data itself
// The prefix is a fixed length byte that states how much data is being transferred over
func createTcpBuffer(data []byte) []byte {
	// Create a buffer with size enough to hold a prefix and actual data
	buf := make([]byte, prefixSize+len(data))

	// State the total number of bytes (including prefix) to be transferred over
	binary.BigEndian.PutUint32(buf[:prefixSize], uint32(prefixSize+len(data)))
	// copy(buf[:prefixSize],[]byte(string(prefixSize+len(data)))
	// Copy data into the remaining buffer
	copy(buf[prefixSize:], data[:])
	return buf
}
