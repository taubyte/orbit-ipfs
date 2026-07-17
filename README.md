# orbit-ipfs

A standalone [Taubyte](https://taubyte.com) **vm-orbit satellite** that exposes
the IPFS host functions to the Taubyte VM, backed by its **own embedded
libp2p + IPFS node**.

Where the in-tree low-orbit IPFS plugin borrows the substrate's `ipfs.Service`,
this satellite owns a full `peer.Node` (libp2p host + boxo DAG/UnixFS/bitswap)
and speaks to the VM over the vm-orbit satellite gRPC SDK
(`github.com/taubyte/tau/pkg/vm-orbit/satellite`). Guest wasm modules see the
same `"ipfs"` module with the same 9 functions and identical semantics.

## Exported host functions

Registered under the wasm module `ipfs`:

- `newIpfsClient`
- `ipfsNewContent`
- `ipfsOpenFile`
- `ipfsCloseFile`
- `ipfsFileCid`
- `ipfsWriteFile`
- `ipfsReadFile`
- `ipfsPushFile`
- `ipfsSeekFile`

Each satellite method is named `W_<Name>`; the `W_` prefix is stripped when the
method is exported to wasm as `<Name>`.

## Backend

On startup the satellite constructs a public `peer.Node` (via
`peer.NewPublic`). `ipfsPushFile` adds a file with `Node.AddFileForCid` and
`ipfsOpenFile` fetches one with `Node.GetFileFromCid`. If no identity is
supplied a fresh one is generated with `keypair.NewRaw()`.

## Configuration

| Env var | Meaning | Default |
| --- | --- | --- |
| `ORBIT_IPFS_SWARM_LISTEN` | comma-separated listen multiaddrs | `/ip4/0.0.0.0/tcp/4001` |
| `ORBIT_IPFS_BOOTSTRAP` | comma-separated bootstrap p2p multiaddrs | none (standalone) |
| `ORBIT_IPFS_SWARM_KEY` | private-network (PSK) swarm key contents | none (public network) |

## Build & run

```sh
go build ./...
```

The produced binary is a hashicorp/go-plugin server. It is launched by the
Taubyte VM as a vm-orbit satellite plugin rather than run directly:

```sh
ORBIT_IPFS_SWARM_LISTEN=/ip4/0.0.0.0/tcp/4001 ./orbit-ipfs
```
