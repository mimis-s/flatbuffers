package codec

// 方便外部直接使用的调用方法

var flatbuffersCodec FlatbuffersCodec

func Marshal(v interface{}) ([]byte, error) {
	return flatbuffersCodec.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return flatbuffersCodec.Unmarshal(data, v)
}

func String() string {
	return flatbuffersCodec.String()
}

func Name() string {
	return flatbuffersCodec.Name()
}
