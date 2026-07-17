// Package backend wraps an embedded libp2p+IPFS node (a taubyte peer.Node)
// and exposes just the two operations the IPFS host functions need:
// adding a file (returning its CID) and fetching a file by CID.
package backend

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ipfs/go-cid"
	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/taubyte/tau/p2p/keypair"
	"github.com/taubyte/tau/p2p/peer"
)

// Backend is the minimal surface the IPFS host functions rely on.
type Backend interface {
	// AddFile stores the contents of r and returns the resulting CID.
	AddFile(r io.Reader) (cid.Cid, error)
	// GetFile returns a reader over the file identified by c.
	GetFile(ctx context.Context, c cid.Cid) (peer.ReadSeekCloser, error)
	// Node exposes the underlying peer.Node.
	Node() peer.Node
	// Close shuts the embedded node down.
	Close()
}

type backend struct {
	node peer.Node
}

// Config controls how the embedded node is constructed. Zero values fall back
// to sane defaults.
type Config struct {
	// PrivateKey is a marshaled libp2p private key. If nil a fresh identity is
	// generated with keypair.NewRaw().
	PrivateKey []byte
	// SwarmKey is an optional private-network (PSK) swarm key. If nil the node
	// joins the public network.
	SwarmKey []byte
	// SwarmListen are the multiaddrs the node listens on.
	SwarmListen []string
	// Bootstrap are the peers to bootstrap against. If empty the node runs
	// standalone.
	Bootstrap []libp2ppeer.AddrInfo
}

// FromEnv builds a Config from ORBIT_IPFS_* environment variables.
//
//	ORBIT_IPFS_SWARM_LISTEN  comma-separated listen multiaddrs
//	                         (default /ip4/0.0.0.0/tcp/4001)
//	ORBIT_IPFS_BOOTSTRAP     comma-separated bootstrap p2p multiaddrs
//	ORBIT_IPFS_SWARM_KEY     private-network swarm key (raw contents)
func FromEnv() (Config, error) {
	cfg := Config{
		SwarmListen: []string{"/ip4/0.0.0.0/tcp/4001"},
	}

	if v := strings.TrimSpace(os.Getenv("ORBIT_IPFS_SWARM_LISTEN")); v != "" {
		cfg.SwarmListen = splitAndTrim(v)
	}

	if v := strings.TrimSpace(os.Getenv("ORBIT_IPFS_SWARM_KEY")); v != "" {
		cfg.SwarmKey = []byte(v)
	}

	if v := strings.TrimSpace(os.Getenv("ORBIT_IPFS_BOOTSTRAP")); v != "" {
		peers, err := parseBootstrap(v)
		if err != nil {
			return cfg, fmt.Errorf("parsing ORBIT_IPFS_BOOTSTRAP failed: %w", err)
		}
		cfg.Bootstrap = peers
	}

	return cfg, nil
}

// New constructs the embedded peer.Node and wraps it as a Backend.
func New(ctx context.Context, cfg Config) (Backend, error) {
	priv := cfg.PrivateKey
	if priv == nil {
		priv = keypair.NewRaw()
		if priv == nil {
			return nil, fmt.Errorf("generating node identity failed")
		}
	}

	listen := cfg.SwarmListen
	if len(listen) == 0 {
		listen = []string{"/ip4/0.0.0.0/tcp/4001"}
	}

	bootstrap := peer.StandAlone()
	if len(cfg.Bootstrap) > 0 {
		bootstrap = peer.Bootstrap(cfg.Bootstrap...)
	}

	// repoPath nil => an ephemeral temp repo the node cleans up on Close.
	node, err := peer.NewPublic(ctx, nil, priv, cfg.SwarmKey, listen, nil, bootstrap)
	if err != nil {
		return nil, fmt.Errorf("creating embedded ipfs node failed: %w", err)
	}

	return &backend{node: node}, nil
}

func (b *backend) AddFile(r io.Reader) (cid.Cid, error) {
	return b.node.AddFileForCid(r)
}

func (b *backend) GetFile(ctx context.Context, c cid.Cid) (peer.ReadSeekCloser, error) {
	return b.node.GetFileFromCid(ctx, c)
}

func (b *backend) Node() peer.Node {
	return b.node
}

func (b *backend) Close() {
	b.node.Close()
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseBootstrap(s string) ([]libp2ppeer.AddrInfo, error) {
	var addrs []ma.Multiaddr
	for _, p := range splitAndTrim(s) {
		m, err := ma.NewMultiaddr(p)
		if err != nil {
			return nil, fmt.Errorf("invalid multiaddr %q: %w", p, err)
		}
		addrs = append(addrs, m)
	}

	return libp2ppeer.AddrInfosFromP2pAddrs(addrs...)
}
