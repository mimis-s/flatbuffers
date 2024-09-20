package codec

import flatbuffers "github.com/google/flatbuffers/go"

var Codec = "flatbuffers"

type FlatTestInterface interface {
	MarshalTable(buf []byte) error
	Pack(builder *flatbuffers.Builder) flatbuffers.UOffsetT
}

type FlatbuffersCodec struct{}

func (FlatbuffersCodec) Marshal(v interface{}) ([]byte, error) {
	fb := flatbuffers.NewBuilder(0)
	fb.Finish(v.(FlatTestInterface).Pack(fb))
	return fb.FinishedBytes(), nil
}

func (FlatbuffersCodec) Unmarshal(data []byte, v interface{}) error {
	return v.(FlatTestInterface).MarshalTable(data)
}

// String  old gRPC Codec interface func
func (FlatbuffersCodec) String() string {
	return Codec
}

// Name returns the name of the Codec implementation. The returned string
// will be used as part of content type in transmission.  The result must be
// static; the result cannot change between calls.
//
// add Name() for ForceCodec interface
func (FlatbuffersCodec) Name() string {
	return Codec
}
