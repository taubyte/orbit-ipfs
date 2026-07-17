package main

import (
	"context"
	"log"

	"github.com/taubyte/orbit-ipfs/backend"
	"github.com/taubyte/orbit-ipfs/ipfs"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := backend.FromEnv()
	if err != nil {
		log.Fatalf("orbit-ipfs: reading configuration failed: %v", err)
	}

	b, err := backend.New(ctx, cfg)
	if err != nil {
		log.Fatalf("orbit-ipfs: creating embedded ipfs node failed: %v", err)
	}
	defer b.Close()

	ipfs.New(ctx, b).Export()
}
