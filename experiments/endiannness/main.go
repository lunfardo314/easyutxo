package main

import (
	"encoding/binary"
	"fmt"
	"math"
)

func main() {
	fmt.Printf("MaxUint32 = %020d\n", math.MaxUint32)

	nuint16 := uint16(0x0201)
	nuint32 := uint32(0x04030201)
	nuint64 := uint64(0x0807060504030201)

	var nb16 [2]byte
	var nb32 [4]byte
	var nb64 [8]byte

	fmt.Printf("--- little-endian: byte order starts from least significant byte ---\n")
	binary.LittleEndian.PutUint16(nb16[:], nuint16)
	fmt.Printf("little-endian 16 dec=%d, hex=0x%x, bin=%+v\n", nuint16, nuint16, nb16)

	binary.LittleEndian.PutUint32(nb32[:], nuint32)
	fmt.Printf("little-endian 32 dec=%d, hex=0x%x, bin=%+v\n", nuint32, nuint32, nb32)

	binary.LittleEndian.PutUint64(nb64[:], nuint64)
	fmt.Printf("little-endian 64 dec=%d, hex=0x%x, bin=%+v\n", nuint64, nuint64, nb64)

	fmt.Printf("\n--- big-endian: byte order starts from most significant byte ---\n")
	binary.BigEndian.PutUint16(nb16[:], nuint16)
	fmt.Printf("big-endian 16 dec=%d, hex=0x%x, bin=%+v\n", nuint16, nuint16, nb16)

	binary.BigEndian.PutUint32(nb32[:], nuint32)
	fmt.Printf("big-endian 32 dec=%d, hex=0x%x, bin=%+v\n", nuint32, nuint32, nb32)

	binary.BigEndian.PutUint64(nb64[:], nuint64)
	fmt.Printf("big-endian 64 dec=%d, hex=0x%x, bin=%+v\n", nuint64, nuint64, nb64)

}

func experimentNil() {
	var d []byte
	if d == nil {
		fmt.Printf("d == nil\n")
	}
	d = []byte{}
	if d == nil {
		fmt.Printf("d = []byte{} == nil\n")
	}
	if f() == nil {
		fmt.Printf("f() == nil\n")
	}
	if g() == nil {
		fmt.Printf("g() == nil\n")
	} else {
		fmt.Printf("g() != nil\n")
	}
	if len(f()) == 0 {
		fmt.Printf("len(f()) == 0\n")
	}
	if len(g()) == 0 {
		fmt.Printf("len(g()) == 0\n")
	}
}

func f() []byte {
	return nil
}

func g() []byte {
	return []byte{}
}
