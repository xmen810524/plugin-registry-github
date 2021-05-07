package github

import (
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type data struct{}

func (d data) MarshalYAML() (interface{}, error) {
	return nil, errors.New("marshal error")
}

func TestYamlReader(t *testing.T) {
	t.Parallel()

	r := newYamlReader(42)

	data, err := afero.ReadAll(r)
	assert.NoError(t, err)

	expected := "42\n"

	assert.Equal(t, expected, string(data))
}

func TestYamlReader_Error(t *testing.T) {
	t.Parallel()

	r := newYamlReader(data{})

	data, err := afero.ReadAll(r)

	expectedError := `marshal error`

	assert.Equal(t, []byte{}, data)
	assert.EqualError(t, err, expectedError)
}
