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
		FromRef:        "nginx:1.27",
		ToRef:          "nginx:1.27",
		FromDigest:     "sha256:aaa",
		ToDigest:       "sha256:bbb",
		FromImageId:    "sha256:111",
		ToImageId:      "sha256:222",
		OldStartedAt:   1_700_000_000_000_000_000,
		At:             1_700_000_100_000_000_000,
		Source:         "schedule",
		RunId:          "run-1",
		Consent:        "schedule",
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

func TestImageSnapshotRoundTrip(t *testing.T) {
	in := &ToolResponse{Type: &ToolResponse_ImageSnapshot{ImageSnapshot: &ImageSnapshot{
		Entries: []*ImageSnapshotEntry{
			{Host: "host-1", Name: "web", ContainerId: "c1", ImageId: "sha256:111", Digest: "sha256:aaa", Ref: "nginx:1.27", StartedAt: 1_700_000_000_000_000_000},
			{Host: "host-1", Name: "local-build", ContainerId: "c2", ImageId: "sha256:333", Ref: "app:dev"},
		},
	}}}

	b, err := proto.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	num, typ, _ := protowire.ConsumeTag(b)
	if num != 7 || typ != protowire.BytesType {
		t.Fatalf("image_snapshot must be ToolResponse field 7, got %d", num)
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
	want := map[string]int32{
		"host": 1, "name": 2, "old_container_id": 3, "new_container_id": 4,
		"from_ref": 5, "to_ref": 6, "from_digest": 7, "to_digest": 8,
		"from_image_id": 9, "to_image_id": 10, "old_started_at": 11, "at": 12,
		"source": 13, "run_id": 14, "consent": 15, "rolled_back": 16,
	}
	fields := (&ContainerUpdate{}).ProtoReflect().Descriptor().Fields()
	if fields.Len() != len(want) {
		t.Fatalf("ContainerUpdate has %d fields, want %d", fields.Len(), len(want))
	}
	for name, num := range want {
		f := fields.ByName(protoreflect.Name(name))
		if f == nil {
			t.Fatalf("missing field %s", name)
		}
		if int32(f.Number()) != num {
			t.Errorf("%s is field %d, want %d", name, f.Number(), num)
		}
	}
}
