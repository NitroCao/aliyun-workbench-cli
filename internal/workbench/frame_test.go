package workbench

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeWBUsesJavaScriptStringLength(t *testing.T) {
	t.Parallel()

	got := encodeWB("text", []byte("测试名"), []byte("😀"))
	want := []byte("4.text,3.测试名,2.😀;")
	assert.Equal(t, want, got)
}

func TestParseWBUsesJavaScriptStringLength(t *testing.T) {
	t.Parallel()

	msgs, err := parseWB([]byte("4.text,3.测试名,2.😀;"))
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "text", msgs[0].Command)
	assert.Equal(t, [][]byte{[]byte("测试名"), []byte("😀")}, msgs[0].Parts)
}
