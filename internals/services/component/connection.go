package component

import (
	"net"
	"sync"
)

// ConnectionPool manages active net.Conn connections mapped to their component ID.
type ConnectionPool struct {
	connections map[string]net.Conn
	mu          sync.RWMutex
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]net.Conn),
	}
}

func (p *ConnectionPool) Add(componentID string, conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connections[componentID] = conn
}

func (p *ConnectionPool) Remove(componentID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.connections, componentID)
}

func (p *ConnectionPool) Get(componentID string) (net.Conn, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	conn, exists := p.connections[componentID]
	return conn, exists
}
