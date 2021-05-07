package github

import (
	"bytes"
	"io"

	"gopkg.in/yaml.v3"
)

type yamlReader struct {
	buffer *bytes.Buffer
	value  interface{}
}

// Read satisfies io.Reader.
func (r *yamlReader) Read(p []byte) (n int, err error) {
	if r.value == nil {
		return 0, io.EOF
	}

	defer func() {
		r.value = nil
	}()

	enc := yaml.NewEncoder(r.buffer)

	if err := enc.Encode(r.value); err != nil {
		return 0, err
	}

	return r.buffer.Read(p)
}

func newYamlReader(v interface{}) *yamlReader {
	return &yamlReader{
		buffer: new(bytes.Buffer),
		value:  v,
	}
}
