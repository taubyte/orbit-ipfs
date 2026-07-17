package ipfs

import (
	"io"

	"github.com/ipfs/go-cid"
	"github.com/taubyte/go-sdk/errno"
	satellite "github.com/taubyte/tau/pkg/vm-orbit/satellite"
)

// cidSize matches the fixed-width CID buffer the guest allocates.
const cidSize = 64

// writeUint32Le writes v as a little-endian uint32 at ptr.
func writeUint32Le(module satellite.Module, ptr, v uint32) errno.Error {
	if _, err := module.WriteUint32(ptr, v); err != nil {
		return errno.ErrorMemoryWriteFailed
	}
	return 0
}

// writeBytes writes the raw bytes at ptr (no length prefix), matching the
// original helpers.WriteBytes semantics (nil becomes a single zero byte).
func writeBytes(module satellite.Module, ptr uint32, value []byte) errno.Error {
	if value == nil {
		value = make([]byte, 1)
	}
	if _, err := module.MemoryWrite(ptr, value); err != nil {
		return errno.ErrorAddressOutOfMemory
	}
	return 0
}

// readCid reads a fixed-width CID from guest memory at ptr.
func readCid(module satellite.Module, ptr uint32) (cid.Cid, errno.Error) {
	cidBytes, err := module.MemoryRead(ptr, cidSize)
	if err != nil {
		return cid.Cid{}, errno.ErrorAddressOutOfMemory
	}

	_, _cid, err := cid.CidFromBytes(cidBytes)
	if err != nil {
		return cid.Cid{}, errno.ErrorInvalidCid
	}

	return _cid, 0
}

// writeCid validates value then writes its byte representation at ptr.
func writeCid(module satellite.Module, ptr uint32, value cid.Cid) errno.Error {
	_cid, err := cid.Parse(value)
	if err != nil {
		return errno.ErrorInvalidCid
	}

	return writeBytes(module, ptr, _cid.Bytes())
}

// read reads up to bufSize bytes from readMethod into the guest buffer at
// bufPtr and writes the count at countPtr, mirroring helpers.Read's errno
// semantics (EOF surfaces as errno.ErrorEOF).
func read(
	module satellite.Module,
	readMethod func(p []byte) (n int, err error),
	bufPtr, bufSize, countPtr uint32,
) errno.Error {
	buf := make([]byte, bufSize)

	n, err := readMethod(buf)
	if err != nil && err != io.EOF {
		return errno.ErrorHttpReadBody
	}

	if _, werr := module.WriteUint32(countPtr, uint32(n)); werr != nil {
		return errno.ErrorAddressOutOfMemory
	}

	if _, werr := module.MemoryWrite(bufPtr, buf); werr != nil {
		return errno.ErrorAddressOutOfMemory
	}

	if err == io.EOF {
		return errno.ErrorEOF
	}

	return 0
}
