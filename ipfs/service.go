// Package ipfs implements the IPFS host functions exported to the Taubyte VM
// as a vm-orbit satellite, backed by an embedded libp2p+IPFS node.
package ipfs

import (
	"context"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/taubyte/go-sdk/errno"
	"github.com/taubyte/orbit-ipfs/backend"
	satellite "github.com/taubyte/tau/pkg/vm-orbit/satellite"
)

// Service is the satellite impl. Methods with the W_ prefix are exported to the
// wasm module "ipfs".
type Service struct {
	backend backend.Backend
	ctx     context.Context

	clients         map[uint32]*client
	clientsLock     sync.RWMutex
	clientsIdToGrab uint32
}

type client struct {
	id              uint32
	contentIdToGrab uint32
	contentLock     sync.RWMutex
	contents        map[uint32]*content
}

type content struct {
	id   uint32
	cid  cid.Cid
	file interface{}
}

// New builds a Service backed by b.
func New(ctx context.Context, b backend.Backend) *Service {
	return &Service{
		backend: b,
		ctx:     ctx,
		clients: make(map[uint32]*client),
	}
}

// Export serves the satellite over the vm-orbit plugin protocol.
func (s *Service) Export() {
	satellite.Export("ipfs", s)
}

func (s *Service) generateClientId() uint32 {
	s.clientsLock.Lock()
	defer func() {
		s.clientsIdToGrab++
		s.clientsLock.Unlock()
	}()
	return s.clientsIdToGrab
}

func (s *Service) getClient(id uint32) (*client, errno.Error) {
	s.clientsLock.RLock()
	c, ok := s.clients[id]
	s.clientsLock.RUnlock()
	if !ok {
		return nil, errno.ErrorClientNotFound
	}
	return c, 0
}

func (s *Service) getClientAndContent(clientId, contentId uint32) (*client, *content, errno.Error) {
	c, err := s.getClient(clientId)
	if err != 0 {
		return nil, nil, err
	}

	ct, err := c.getContent(contentId)
	if err != 0 {
		return c, nil, err
	}

	return c, ct, 0
}

func (c *client) generateContentId() uint32 {
	c.contentLock.Lock()
	defer func() {
		c.contentIdToGrab++
		c.contentLock.Unlock()
	}()
	return c.contentIdToGrab
}

func (c *client) generateContent(id uint32, _cid cid.Cid, file interface{}) *content {
	ct := &content{id: id, cid: _cid, file: file}

	c.contentLock.Lock()
	c.contents[id] = ct
	c.contentLock.Unlock()

	return ct
}

func (c *client) getContent(contentId uint32) (*content, errno.Error) {
	c.contentLock.RLock()
	ct, ok := c.contents[contentId]
	c.contentLock.RUnlock()
	if !ok {
		return nil, errno.ErrorContentNotFound
	}
	return ct, 0
}

// W_newIpfsClient creates a new client and writes its id at clientIdPtr.
func (s *Service) W_newIpfsClient(ctx context.Context, module satellite.Module,
	clientIdPtr uint32,
) uint32 {
	c := &client{
		id:       s.generateClientId(),
		contents: make(map[uint32]*content),
	}

	s.clientsLock.Lock()
	s.clients[c.id] = c
	s.clientsLock.Unlock()

	return uint32(writeUint32Le(module, clientIdPtr, c.id))
}
