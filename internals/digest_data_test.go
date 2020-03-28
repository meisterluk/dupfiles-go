package internals

import "testing"

func TestAdd(t *testing.T) {
	digest1 := []byte{0x42, 0x41, 0x66, 0xFF}
	digest2 := []byte{0x42, 0x41, 0x99, 0xFF}
	digest3 := []byte{0x00, 0x41, 0x66, 0xFF}

	dd := NewDigestData(4, 2)
	index, found := dd.Add(digest1)
	if index != 0 {
		t.Errorf("expected new index 0, got %d", index)
	}
	if found {
		t.Errorf("expected new digest to be missing, found it")
	}

	index2, found := dd.Add(digest2)
	if index2 != 1 {
		t.Errorf("expected new index 1, got %d", index2)
	}
	if found {
		t.Errorf("expected new digest to be missing, found it")
	}

	index3, found := dd.Add(digest1)
	if index3 != 0 {
		t.Errorf("expected existing index 0, got %d", index3)
	}
	if !found {
		t.Errorf("expected old digest to exist, missing it")
	}

	index4, found := dd.Add(digest3)
	if index4 != 0 {
		t.Errorf("expected existing index 0, got %d", index4)
	}
	if found {
		t.Errorf("expected new digest to be missing, found it")
	}
}
