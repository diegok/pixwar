package protocol

import (
	"encoding/gob"
	"io"
)

// Codec handles message encoding/decoding
type Codec struct {
	enc *gob.Encoder
	dec *gob.Decoder
}

// NewCodec creates a codec for the given read/writer
func NewCodec(rw io.ReadWriter) *Codec {
	return &Codec{
		enc: gob.NewEncoder(rw),
		dec: gob.NewDecoder(rw),
	}
}

// NewEncoder creates an encoder-only codec
func NewEncoder(w io.Writer) *Codec {
	return &Codec{
		enc: gob.NewEncoder(w),
	}
}

// NewDecoder creates a decoder-only codec
func NewDecoder(r io.Reader) *Codec {
	return &Codec{
		dec: gob.NewDecoder(r),
	}
}

// Encode writes a message
func (c *Codec) Encode(msg *Message) error {
	return c.enc.Encode(msg)
}

// Decode reads a message
func (c *Codec) Decode() (*Message, error) {
	var msg Message
	if err := c.dec.Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
