package fastlike

import (
	"bytes"
	"io"
)

func (i *Instance) xqd_body_new(handle_out int32) int32 {
	bhid, _ := i.bodies.NewBuffer()
	i.abilog.Printf("body_new: handle=%d", bhid)
	i.memory.PutUint32(uint32(bhid), int64(handle_out))
	return int32(XqdStatusOK)
}

func (i *Instance) xqd_body_write(handle int32, addr int32, size int32, body_end int32, nwritten_out int32) int32 {
	// TODO: Figure out what we're supposed to do with `body_end` which can be 0 (back) or
	// 1 (front)
	i.abilog.Printf("body_write: handle=%d size=%d, body_end=%d", handle, size, body_end)

	body := i.bodies.Get(int(handle))
	if body == nil {
		return int32(XqdErrInvalidHandle)
	}

	// Copy size bytes starting at addr into the body handle
	nwritten, err := io.CopyN(body, bytes.NewReader(i.memory.Data()[addr:]), int64(size))
	if err != nil {
		// TODO: If err == EOF then there's a specific error code we can return (it means they
		// didn't have `size` bytes in memory)
		return int32(XqdError)
	}

	// Write out how many bytes we copied
	i.memory.PutUint32(uint32(nwritten), int64(nwritten_out))

	return int32(XqdStatusOK)
}

func (i *Instance) xqd_body_read(handle int32, addr int32, maxlen int32, nread_out int32) int32 {
	body := i.bodies.Get(int(handle))
	if body == nil {
		return int32(XqdErrInvalidHandle)
	}

	buf := bytes.NewBuffer(make([]byte, 0, maxlen))
	ncopied, err := io.Copy(buf, io.LimitReader(body, int64(maxlen)))
	if err != nil {
		i.abilog.Printf("body_read: error copying got=%s", err.Error())
		return int32(XqdError)
	}

	nwritten, err2 := i.memory.WriteAt(buf.Bytes(), int64(addr))
	if err2 != nil {
		i.abilog.Printf("body_read: error writing got=%s", err2.Error())
		return int32(XqdError)
	}

	if ncopied != int64(nwritten) {
		i.abilog.Printf("body_read: error copying copied=%d wrote=%d", ncopied, nwritten)
		return int32(XqdError)
	}

	i.abilog.Printf("body_read: handle=%d copied=%d", handle, ncopied)

	// Write out how many bytes we copied
	i.memory.PutUint32(uint32(nwritten), int64(nread_out))

	return int32(XqdStatusOK)
}

func (i *Instance) xqd_body_append(dst_handle int32, src_handle int32) int32 {
	i.abilog.Printf("body_append: dst=%d src=%d", dst_handle, src_handle)

	dst := i.bodies.Get(int(dst_handle))
	if dst == nil {
		return int32(XqdErrInvalidHandle)
	}

	src := i.bodies.Get(int(src_handle))
	if src == nil {
		return int32(XqdErrInvalidHandle)
	}

	// replace the destination reader with a multireader that reads first from the original reader
	// and then from the source
	dst.reader = io.MultiReader(dst.reader, src)

	return int32(XqdStatusOK)
}
