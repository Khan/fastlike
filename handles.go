package fastlike

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

type fastlyMeta struct{}

// RequestHandle is an http.Request with extra metadata
// Notably, the request body is ignored and instead the guest will provide a BodyHandle to use
type RequestHandle struct {
	*http.Request
	fastlyMeta *fastlyMeta

	// It is an error to try sending a request without an associated body handle
	hasBody bool
}

// RequestHandles is a slice of RequestHandle with functions to get and create
type RequestHandles struct {
	handles []*RequestHandle
}

// Get returns the RequestHandle identified by id or nil if one does not exist.
func (rhs *RequestHandles) Get(id int) *RequestHandle {
	if id >= len(rhs.handles) {
		return nil
	}

	return rhs.handles[id]
}

// New creates a new RequestHandle and returns its handle id and the handle itself.
func (rhs *RequestHandles) New() (int, *RequestHandle) {
	rh := &RequestHandle{Request: &http.Request{}}
	rhs.handles = append(rhs.handles, rh)
	return len(rhs.handles) - 1, rh
}

// ResponseHandle is an http.Response with extra metadata
// Notably, the response body is ignored and instead the guest will provide a BodyHandle to use
type ResponseHandle struct {
	*http.Response

	// It is an error to try sending a response without an associated body handle
	hasBody bool
}

// ResponseHandles is a slice of ResponseHandle with functions to get and create
type ResponseHandles struct {
	handles []*ResponseHandle
}

// Get returns the ResponseHandle identified by id or nil if one does not exist.
func (rhs *ResponseHandles) Get(id int) *ResponseHandle {
	if id >= len(rhs.handles) {
		return nil
	}

	return rhs.handles[id]
}

// New creates a new ResponseHandle and returns its handle id and the handle itself.
func (rhs *ResponseHandles) New() (int, *ResponseHandle) {
	rh := &ResponseHandle{Response: &http.Response{StatusCode: 200}}
	rhs.handles = append(rhs.handles, rh)
	return len(rhs.handles) - 1, rh
}

// BodyHandle represents a body. It could be readable or writable, but not both.
// For cases where it's already connected to a request or response body, the reader or writer
// properties will reference the original request or response respectively.
// For new bodies, buf will hold the contents and either the reader or writer will wrap it.
type BodyHandle struct {

	// reader, writer, and closer are connected to the existing request/response body, if one exists
	reader io.Reader
	writer io.Writer
	closer io.Closer

	// for bodies created outside of a request/response, buf holds the body content and
	// reader/writer/closer wrap it
	buf *bytes.Buffer

	// length is the number of bytes in the body
	length int64
}

// Close implements io.Closer for a BodyHandle
func (b *BodyHandle) Close() error {
	if b.closer != nil {
		return b.closer.Close()
	}
	return nil
}

// Read implements io.Reader for a BodyHandle
func (b *BodyHandle) Read(p []byte) (int, error) {
	return b.reader.Read(p)
}

// Write implements io.Writer for a BodyHandle
func (b *BodyHandle) Write(p []byte) (int, error) {
	n, e := b.writer.Write(p)
	b.length += int64(n)
	return n, e
}

func (b *BodyHandle) Size() int64 {
	if b.length == 0 {
		return -1
	}
	return b.length
}

// BodyHandles is a slice of BodyHandle with methods to get and create
type BodyHandles struct {
	lock         sync.RWMutex
	nextHandleID int
	handles      map[int]*BodyHandle
}

func NewBodyHandles() *BodyHandles {
	return &BodyHandles{handles: make(map[int]*BodyHandle)}
}

func (bhs *BodyHandles) addBodyHandle(bh *BodyHandle) (int, *BodyHandle) {
	bhs.lock.Lock()
	defer bhs.lock.Unlock()

	id := bhs.nextHandleID
	bhs.nextHandleID += 1

	bhs.handles[id] = bh

	return id, bh
}

// Get returns the BodyHandle identified by id or nil if one does not exist
func (bhs *BodyHandles) Get(id int) *BodyHandle {
	bhs.lock.RLock()
	defer bhs.lock.RUnlock()

	return bhs.handles[id]
}

// NewBuffer creates a BodyHandle backed by a buffer which can be read from or written to
func (bhs *BodyHandles) NewBuffer() (int, *BodyHandle) {
	bh := &BodyHandle{buf: new(bytes.Buffer)}
	bh.reader = io.Reader(bh.buf)
	bh.writer = io.Writer(bh.buf)

	return bhs.addBodyHandle(bh)
}

// NewReader creates a BodyHandle whose reader and closer is connected to the supplied ReadCloser
func (bhs *BodyHandles) NewReader(rdr io.ReadCloser) (int, *BodyHandle) {
	bh := &BodyHandle{}
	bh.reader = rdr
	bh.closer = rdr
	bh.writer = ioutil.Discard

	return bhs.addBodyHandle(bh)
}

// NewWriter creates a BodyHandle whose writer is connected to the supplied Writer
func (bhs *BodyHandles) NewWriter(w io.Writer) (int, *BodyHandle) {
	bh := &BodyHandle{}
	bh.writer = w

	return bhs.addBodyHandle(bh)
}

func (bhs *BodyHandles) Close(id int) error {
	bhs.lock.Lock()
	defer bhs.lock.Unlock()

	// We can't use .Get() here because that'd cause a deadlock on the RWMutex
	bh := bhs.handles[id]
	if bh == nil {
		return nil
	}

	delete(bhs.handles, id)

	return bh.Close()
}
