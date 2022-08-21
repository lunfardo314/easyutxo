package proto

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestProto(t *testing.T) {
	// [START populate_proto]
	p := Person{
		Id:    1234,
		Name:  "John Doe",
		Email: "jdoe@example.com",
		Phones: []*Person_PhoneNumber{
			{Number: "555-4321", Type: Person_HOME},
		},
	}
	bin, err := proto.Marshal(&p)
	require.NoError(t, err)
	t.Logf("serialized len bin: %d", len(bin))
	pBack := &Person{}
	err = proto.Unmarshal(bin, pBack)
	require.NoError(t, err)

}
