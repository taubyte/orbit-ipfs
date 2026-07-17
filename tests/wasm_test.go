package tests

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/taubyte/tau/pkg/vm-orbit/tests/suite"
	builder "github.com/taubyte/tau/pkg/vm-orbit/tests/suite/builders/go"
	"gotest.tools/v3/assert"
)

// TestIpfsRoundTrip builds this repo's satellite plugin and a guest wasm module
// (from _fixtures/dfunc.go, which uses github.com/taubyte/go-sdk/ipfs), attaches
// the plugin to a testing vm suite, and drives a full add->get round-trip.
//
// The satellite embeds its own libp2p+IPFS node and reads ORBIT_IPFS_* env at
// startup. We force it standalone + ephemeral BEFORE attaching the plugin so the
// round-trip is served entirely from the node's local blockstore — no network,
// no peers, hermetic and fast. hashicorp/go-plugin launches the satellite with
// os.Environ() appended, so these t.Setenv values propagate to the subprocess.
func TestIpfsRoundTrip(t *testing.T) {
	// Standalone (offline) + ephemeral ipfs node inside the satellite subprocess.
	t.Setenv("ORBIT_IPFS_BOOTSTRAP", "none")
	t.Setenv("ORBIT_IPFS_SWARM_LISTEN", "/ip4/127.0.0.1/tcp/0")

	ctx := context.Background()

	testingSuite, err := suite.New(ctx)
	assert.NilError(t, err)
	defer testingSuite.Close()

	goBuilder := builder.New()

	wd, err := os.Getwd()
	assert.NilError(t, err)

	// Build the satellite plugin from the module root (where main.go lives).
	pluginPath, err := goBuilder.Plugin(path.Join(wd, ".."), "ipfs")
	assert.NilError(t, err)

	// Attach it: this launches the satellite subprocess, which now reads the
	// ORBIT_IPFS_* env set above.
	err = testingSuite.AttachPluginFromPath(pluginPath)
	assert.NilError(t, err)

	// Build the guest wasm from our fixture.
	wasmPath, err := goBuilder.Wasm(ctx, path.Join(wd, "_fixtures", "dfunc.go"))
	assert.NilError(t, err)

	module, err := testingSuite.WasmModule(wasmPath)
	assert.NilError(t, err)

	// Drive the round-trip exported by the fixture.
	_, err = module.Call(ctx, "roundtrip")
	assert.NilError(t, err)

	// The fixture prints this on a successful verified round-trip.
	module.AssetOutput(t, "roundtrip-ok\n")
}
