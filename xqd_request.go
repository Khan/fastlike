package fastlike

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
)

func (i *Instance) xqd_req_version_get(handle int32, version_out int32) int32 {
	if i.requests.Get(int(handle)) == nil {
		i.abilog.Printf("req_version_get: invalid handle %d", handle)
		return int32(XqdErrInvalidHandle)
	}

	i.abilog.Printf("req_version_get: handle=%d version=%d", handle, Http11)
	i.memory.PutUint32(uint32(Http11), int64(version_out))
	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_version_set(handle int32, version int32) int32 {
	i.abilog.Printf("req_version_set: handle=%d version=%d", handle, version)

	if i.requests.Get(int(handle)) == nil {
		i.abilog.Printf("req_version_set: invalid handle %d", handle)
		return int32(XqdErrInvalidHandle)
	}

	if version != int32(Http11) {
		i.abilog.Printf("req_version_set: invalid version %d", version)
		return int32(XqdErrUnsupported)
	}

	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_cache_override_set(handle int32, tag int32, ttl int32, swr int32) int32 {
	// We don't actually *do* anything with cache overrides, since we don't have or need a cache.

	if i.requests.Get(int(handle)) == nil {
		i.abilog.Printf("req_cache_override_set: invalid handle %d", handle)
		return int32(XqdErrInvalidHandle)
	}

	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_cache_override_set_v2(handle int32, tag int32, ttl int32, swr int32, sk int32, sk_len int32) int32 {
	// We don't actually *do* anything with cache overrides, since we don't have or need a cache.

	if i.requests.Get(int(handle)) == nil {
		i.abilog.Printf("cache_override_v2_set: invalid handle %d", handle)
		return int32(XqdErrInvalidHandle)
	}

	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_method_get(handle int32, addr int32, maxlen int32, nwritten_out int32) int32 {
	r := i.requests.Get(int(handle))
	if r == nil {
		i.abilog.Printf("req_method_get: invalid handle %d", handle)
		return int32(XqdErrInvalidHandle)
	}

	if int(maxlen) < len(r.Method) {
		return int32(XqdErrBufferLength)
	}

	i.abilog.Printf("req_method_get: handle=%d method=%q", handle, r.Method)

	nwritten, err := i.memory.WriteAt([]byte(r.Method), int64(addr))
	if err != nil {
		return int32(XqdError)
	}

	i.memory.PutUint32(uint32(nwritten), int64(nwritten_out))
	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_method_set(handle int32, addr int32, size int32) int32 {
	r := i.requests.Get(int(handle))
	if r == nil {
		return int32(XqdErrInvalidHandle)
	}

	method := make([]byte, size)
	_, err := i.memory.ReadAt(method, int64(addr))
	if err != nil {
		return int32(XqdError)
	}

	// Make sure the method is in the set of valid http methods
	methods := strings.Join([]string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}, "\x00")

	if !strings.Contains(methods, strings.ToUpper(string(method))) {
		i.abilog.Printf("req_method_set: invalid method=%q", method)
		return int32(XqdErrHttpParse)
	}

	i.abilog.Printf("req_method_set: handle=%d method=%q", handle, method)

	r.Method = strings.ToUpper(string(method))
	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_uri_set(handle int32, addr int32, size int32) int32 {
	r := i.requests.Get(int(handle))
	if r == nil {
		return int32(XqdErrInvalidHandle)
	}

	buf := make([]byte, size)
	nread, err := i.memory.ReadAt(buf, int64(addr))
	if err != nil {
		return int32(XqdError)
	}
	if nread < int(size) {
		return int32(XqdError)
	}

	u, err := url.Parse(string(buf))
	if err != nil {
		i.abilog.Printf("req_uri_set: parse error uri=%q got=%s", buf, err.Error())
		return int32(XqdErrHttpParse)
	}

	i.abilog.Printf("req_uri_set: handle=%d uri=%q", handle, u)

	r.URL = u
	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_header_names_get(handle int32, addr int32, maxlen int32, cursor int32, ending_cursor_out int32, nwritten_out int32) int32 {
	i.abilog.Printf("req_header_names_get: handle=%d cursor=%d", handle, cursor)

	r := i.requests.Get(int(handle))
	if r == nil {
		return int32(XqdErrInvalidHandle)
	}

	names := []string{}
	for n := range r.Header {
		names = append(names, n)
	}

	// these names are explicitly unsorted, so let's sort them ourselves
	sort.Strings(names[:])

	return int32(xqd_multivalue(i.memory, names, addr, maxlen, cursor, ending_cursor_out, nwritten_out))
}

func (i *Instance) xqd_req_header_values_get(handle int32, name_addr int32, name_size int32, addr int32, maxlen int32, cursor int32, ending_cursor_out int32, nwritten_out int32) int32 {
	r := i.requests.Get(int(handle))
	if r == nil {
		return int32(XqdErrInvalidHandle)
	}

	buf := make([]byte, name_size)
	_, err := i.memory.ReadAt(buf, int64(name_addr))
	if err != nil {
		return int32(XqdError)
	}

	header := http.CanonicalHeaderKey(string(buf))

	i.abilog.Printf("req_header_values_get: handle=%d header=%q cursor=%d\n", handle, header, cursor)

	values, ok := r.Header[header]
	if !ok {
		values = []string{}
	}

	// Sort the values otherwise cursors don't work
	sort.Strings(values[:])

	return int32(xqd_multivalue(i.memory, values, addr, maxlen, cursor, ending_cursor_out, nwritten_out))
}

func (i *Instance) xqd_req_header_values_set(handle int32, name_addr int32, name_size int32, values_addr int32, values_size int32) int32 {
	r := i.requests.Get(int(handle))
	if r == nil {
		return int32(XqdErrInvalidHandle)
	}

	buf := make([]byte, name_size)
	_, err := i.memory.ReadAt(buf, int64(name_addr))
	if err != nil {
		return int32(XqdError)
	}

	header := http.CanonicalHeaderKey(string(buf))

	// read values_size bytes from values_addr for a list of \0 terminated values for the header
	// but, read 1 less than that to avoid the trailing nul
	buf = make([]byte, values_size-1)
	_, err = i.memory.ReadAt(buf, int64(values_addr))
	if err != nil {
		return int32(XqdError)
	}

	values := bytes.Split(buf, []byte("\x00"))

	i.abilog.Printf("req_header_values_set: handle=%d header=%q values=%q\n", handle, header, values)

	if r.Header == nil {
		r.Header = http.Header{}
	}

	for _, v := range values {
		r.Header.Add(header, string(v))
	}

	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_uri_get(handle int32, addr int32, maxlen int32, nwritten_out int32) int32 {
	r := i.requests.Get(int(handle))
	if r == nil {
		return int32(XqdErrInvalidHandle)
	}

	uri := r.URL.String()
	i.abilog.Printf("req_uri_get: handle=%d uri=%q", handle, uri)

	nwritten, err := i.memory.WriteAt([]byte(uri), int64(addr))
	if err != nil {
		return int32(XqdError)
	}

	i.memory.PutUint32(uint32(nwritten), int64(nwritten_out))
	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_new(handle_out int32) int32 {
	rhid, _ := i.requests.New()
	i.abilog.Printf("req_new: handle=%d", rhid)
	i.memory.PutUint32(uint32(rhid), int64(handle_out))
	return int32(XqdStatusOK)
}

func (i *Instance) xqd_req_send(rhandle int32, bhandle int32, backend_addr, backend_size int32, wh_out int32, bh_out int32) int32 {
	// sends the request described by (rh, bh) to the backend
	// expects a response handle and response body handle
	r := i.requests.Get(int(rhandle))
	if r == nil {
		i.abilog.Printf("req_send: invalid request handle=%d", rhandle)
		return int32(XqdErrInvalidHandle)
	}

	b := i.bodies.Get(int(bhandle))
	if b == nil {
		i.abilog.Printf("req_send: invalid body handle=%d", bhandle)
		return int32(XqdErrInvalidHandle)
	}

	buf := make([]byte, backend_size)
	_, err := i.memory.ReadAt(buf, int64(backend_addr))
	if err != nil {
		return int32(XqdError)
	}

	backend := string(buf)

	i.abilog.Printf("req_send: handle=%d body=%d backend=%q uri=%q", rhandle, bhandle, backend, r.URL)

	req, err := http.NewRequest(r.Method, r.URL.String(), b)
	if err != nil {
		return int32(XqdErrHttpUserInvalid)
	}

	req.Header = r.Header.Clone()

	// TODO: Ensure we always have something in r.Header so we can avoid the nil check here
	if req.Header == nil {
		req.Header = http.Header{}
	}

	// Make sure to add a CDN-Loop header, which we can check (and block) at ingress
	req.Header.Add("cdn-loop", "fastlike")

	// TODO: Not sure if this is strictly necessary (or correct!)
	if req.Header.Get("content-length") == "" {
		req.Header.Add("content-length", fmt.Sprintf("%d", b.Size()))
		req.ContentLength = b.Size()
	} else if req.ContentLength <= 0 {
		req.ContentLength = b.Size()
	}

	// If the backend is geolocation, we select the geobackend explicitly
	var handler http.Handler
	if backend == "geolocation" {
		handler = i.geobackend
	} else {
		handler = i.backends(backend)
	}

	// TODO: Is there a better way to get an *http.Response from an http.Handler?
	// The Handler interface is useful for embedders, since often-times they'll be processing wasm
	// requests in the embedding application, and it's very easy to adapt an http.Handler to an
	// http.RoundTripper if they want it to go offsite.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, req)

	w := wr.Result()

	// Convert the response into an (rh, bh) pair, put them in the list, and write out the handles
	whid, wh := i.responses.New()
	wh.Status = w.Status
	wh.StatusCode = w.StatusCode
	wh.Header = w.Header.Clone()
	wh.Body = w.Body

	bhid, _ := i.bodies.NewReader(wh.Body)

	i.abilog.Printf("req_send: response handle=%d body=%d", whid, bhid)

	i.memory.PutUint32(uint32(whid), int64(wh_out))
	i.memory.PutUint32(uint32(bhid), int64(bh_out))

	return int32(XqdStatusOK)
}
