package ipfs

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/taubyte/go-sdk/errno"
	satellite "github.com/taubyte/tau/pkg/vm-orbit/satellite"
)

// W_ipfsNewContent creates a fresh temp-file backed content and writes its id
// at contentIdPtr.
func (s *Service) W_ipfsNewContent(ctx context.Context, module satellite.Module,
	clientId,
	contentIdPtr uint32,
) uint32 {
	c, err0 := s.getClient(clientId)
	if err0 != 0 {
		return uint32(err0)
	}

	contentId := c.generateContentId()
	newFile, err := os.Create("tempFile" + fmt.Sprint("", contentId))
	if err != nil {
		return uint32(errno.ErrorCreatingNewFile)
	}

	ct := c.generateContent(contentId, cid.Cid{}, newFile)
	return uint32(writeUint32Le(module, contentIdPtr, ct.id))
}

// W_ipfsOpenFile reads a CID from guest memory, fetches the file from the
// embedded node and registers it as a new content.
func (s *Service) W_ipfsOpenFile(ctx context.Context, module satellite.Module,
	clientId,
	contentIdPtr,
	cidPtr uint32,
) uint32 {
	c, err0 := s.getClient(clientId)
	if err0 != 0 {
		return uint32(err0)
	}

	_cid, err0 := readCid(module, cidPtr)
	if err0 != 0 {
		return uint32(err0)
	}

	file, err := s.backend.GetFile(s.ctx, _cid)
	if err != nil {
		return uint32(errno.ErrorCidNotFoundOnIpfs)
	}

	ct := c.generateContent(c.generateContentId(), _cid, file)
	return uint32(writeUint32Le(module, contentIdPtr, ct.id))
}

// W_ipfsCloseFile closes the content's underlying file.
func (s *Service) W_ipfsCloseFile(ctx context.Context, module satellite.Module,
	clientId,
	contentId uint32,
) uint32 {
	_, ct, err0 := s.getClientAndContent(clientId, contentId)
	if err0 != 0 {
		return uint32(err0)
	}

	if err := ct.file.(io.Closer).Close(); err != nil {
		return uint32(errno.ErrorCloseFileFailed)
	}

	return 0
}

// W_ipfsFileCid writes the content's CID at cidPtr.
func (s *Service) W_ipfsFileCid(ctx context.Context, module satellite.Module,
	clientId,
	contentId,
	cidPtr uint32,
) uint32 {
	_, ct, err0 := s.getClientAndContent(clientId, contentId)
	if err0 != 0 {
		return uint32(err0)
	}

	_cid, err := cid.Parse(ct.cid)
	if err != nil {
		return uint32(errno.ErrorInvalidCid)
	}

	return uint32(writeBytes(module, cidPtr, _cid.Bytes()))
}

// W_ipfsWriteFile writes bufLen bytes from the guest buffer into the content
// file and reports how many bytes were written at writePtr.
func (s *Service) W_ipfsWriteFile(ctx context.Context, module satellite.Module,
	clientId,
	contentId,
	buf, bufLen,
	writePtr uint32,
) uint32 {
	data, err := module.MemoryRead(buf, bufLen)
	if err != nil {
		return uint32(errno.ErrorAddressOutOfMemory)
	}

	c, err0 := s.getClient(clientId)
	if err0 != 0 {
		return uint32(err0)
	}

	ct, err0 := c.getContent(contentId)
	if err0 != 0 {
		return uint32(err0)
	}

	written, werr := ct.file.(io.Writer).Write(data)
	if werr != nil {
		return uint32(errno.ErrorWritingFile)
	}

	return uint32(writeUint32Le(module, writePtr, uint32(written)))
}

// W_ipfsReadFile reads up to bufLen bytes from the content file into the guest
// buffer, writing the count at countPtr.
func (s *Service) W_ipfsReadFile(ctx context.Context, module satellite.Module,
	clientId,
	contentId,
	buf, bufLen,
	countPtr uint32,
) uint32 {
	_, ct, err0 := s.getClientAndContent(clientId, contentId)
	if err0 != 0 {
		return uint32(err0)
	}

	return uint32(read(module, ct.file.(io.Reader).Read, buf, bufLen, countPtr))
}

// W_ipfsPushFile seeks the content file to 0, adds it to the embedded node and
// writes the resulting CID at cidPtr.
func (s *Service) W_ipfsPushFile(ctx context.Context, module satellite.Module,
	clientId,
	contentId,
	cidPtr uint32,
) uint32 {
	_, ct, err0 := s.getClientAndContent(clientId, contentId)
	if err0 != 0 {
		return uint32(err0)
	}

	file, ok := ct.file.(io.ReadSeeker)
	if !ok {
		return uint32(errno.ErrorAddFileFailed)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return uint32(errno.ErrorAddFileFailed)
	}

	_cid, err := s.backend.AddFile(file)
	if err != nil {
		return uint32(errno.ErrorAddFileFailed)
	}

	return uint32(writeCid(module, cidPtr, _cid))
}

// W_ipfsSeekFile seeks the content file and writes the new offset at offsetPtr.
func (s *Service) W_ipfsSeekFile(ctx context.Context, module satellite.Module,
	clientId,
	contentId uint32,
	offset int64,
	whence,
	offsetPtr uint32,
) uint32 {
	_, ct, err0 := s.getClientAndContent(clientId, contentId)
	if err0 != 0 {
		return uint32(err0)
	}

	if int(whence) > 2 || int(whence) < 0 {
		return uint32(errno.ErrorInvalidWhence)
	}

	_offset, err := ct.file.(io.Seeker).Seek(offset, int(whence))
	if err != nil {
		return uint32(errno.ErrorSeekingFile)
	}

	return uint32(writeUint32Le(module, offsetPtr, uint32(_offset)))
}
