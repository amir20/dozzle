package cloud

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Cloud decodes these pushes with its own copy of cloud.proto, so the field
// numbers are the contract. These tests pin them and check a full round trip.

func TestContainerUpdateRoundTrip(t *testing.T) {
	in := &ToolResponse{Type: &ToolResponse_ContainerUpdate{ContainerUpdate: &ContainerUpdate{
		Host:           "host-1",
		Name:           "web",
		OldContainerId: "old",
		NewContainerId: "new",
		ImageRef:       "nginx:1.27",
		FromDigest:     "sha256:aaa",
		ToDigest:       "sha256:bbb",
		FromImageId:    "sha256:111",
		ToImageId:      "sha256:222",
		OldStartedAt:   1_700_000_000_000_000_000,
		At:             1_700_000_100_000_000_000,
		Source:         "schedule",
		RolledBack:     true,
	}}}

	b, err := proto.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	num, typ, _ := protowire.ConsumeTag(b)
	if num != 6 || typ != protowire.BytesType {
		t.Fatalf("container_update must be ToolResponse field 6, got %d", num)
	}

	out := &ToolResponse{}
	if err := proto.Unmarshal(b, out); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(in, out) {
		t.Fatalf("round trip changed the message:\n in: %v\nout: %v", in, out)
	}
}

// The field numbers inside ContainerUpdate, by name. A renumbering would make
// Cloud read one field as another without any error.
func TestContainerUpdateFieldNumbers(t *testing.T) {
	assertFieldNumbers(t, &ContainerUpdate{}, map[string]int32{
		"host": 1, "name": 2, "old_container_id": 3, "new_container_id": 4,
		"image_ref": 5, "from_digest": 7, "to_digest": 8,
		"from_image_id": 9, "to_image_id": 10, "old_started_at": 11, "at": 12,
		"source": 13, "rolled_back": 16,
	})
}

func assertFieldNumbers(t *testing.T, m proto.Message, want map[string]int32) {
	t.Helper()
	desc := m.ProtoReflect().Descriptor()
	fields := desc.Fields()
	if fields.Len() != len(want) {
		t.Fatalf("%s has %d fields, want %d", desc.Name(), fields.Len(), len(want))
	}
	for name, num := range want {
		f := fields.ByName(protoreflect.Name(name))
		if f == nil {
			t.Fatalf("%s is missing field %s", desc.Name(), name)
		}
		if int32(f.Number()) != num {
			t.Errorf("%s.%s is field %d, want %d", desc.Name(), name, f.Number(), num)
		}
	}
}
