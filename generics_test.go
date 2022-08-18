package easyutxo

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func testOne[T integerIntern](t *testing.T, v T) {
	var buf bytes.Buffer
	err := WriteInteger(&buf, v)
	require.NoError(t, err)
	b := buf.Bytes()
	require.EqualValues(t, binary.Size(v), len(b))

	var vBack T

	err = ReadInteger(bytes.NewReader(b), &vBack)
	require.NoError(t, err)

	require.True(t, v == vBack)
}

func TestWriteRead(t *testing.T) {
	u8 := uint8(1)
	u16 := uint16(2)
	u32 := uint32(3)
	u64 := uint64(4)
	i8 := int8(-5)
	i16 := int16(-6)
	i32 := int32(-7)
	i64 := int64(-8)

	testOne(t, u8)
	testOne(t, u16)
	testOne(t, u32)
	testOne(t, u64)

	testOne(t, i8)
	testOne(t, i16)
	testOne(t, i32)
	testOne(t, i64)
}
