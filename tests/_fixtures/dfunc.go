package lib

import (
	"fmt"
	"io"

	"github.com/taubyte/go-sdk/ipfs/client"
)

// roundtrip exercises the orbit-ipfs satellite end-to-end from guest wasm:
// create content, write bytes, push to obtain a CID, re-open that CID, read the
// bytes back and verify they match. On success it prints a deterministic line
// to stdout so the host test can assert on it. Any failure panics (surfaced on
// stderr) so the test fails loudly rather than silently mismatching.
//
//export roundtrip
func roundtrip() {
	c, err := client.New()
	if err != nil {
		panic(err)
	}

	content, err := c.Create()
	if err != nil {
		panic(err)
	}

	payload := []byte("orbit-ipfs-roundtrip")
	if _, err := content.Write(payload); err != nil {
		panic(err)
	}

	cid, err := content.Push()
	if err != nil {
		panic(err)
	}

	opened, err := c.Open(cid)
	if err != nil {
		panic(err)
	}

	got, err := io.ReadAll(opened)
	if err != nil {
		panic(err)
	}
	if err := opened.Close(); err != nil {
		panic(err)
	}

	if string(got) != string(payload) {
		panic("roundtrip mismatch: read back " + string(got))
	}

	// fmt.Println writes to stdout (unlike the println builtin, which tinygo
	// routes to stderr) so the host asserts on stdout via AssetOutput.
	fmt.Println("roundtrip-ok")
}
