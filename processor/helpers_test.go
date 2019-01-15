package processor

import (
	"testing"
	"time"
)

func TestMakeTimestampNano(t *testing.T) {
	res := makeTimestampNano()
	time.Sleep(5 * time.Nanosecond)

	if res == makeTimestampNano() {
		t.Error("Should not match")
	}
}

func TestMakeTimestampMilli(t *testing.T) {
	res := makeTimestampMilli()
	time.Sleep(5 * time.Millisecond)

	if res == makeTimestampMilli() {
		t.Error("Should not match")
	}
}

func TestMax(t *testing.T) {
	if 2 != max(1, 2) {
		t.Errorf("Max should be 2")
	}

	if 2 != max(2, 2) {
		t.Errorf("Max should be 2")
	}

	if 3 != max(3, 1) {
		t.Errorf("Max should be 3")
	}

	if 3 != max(1, 3) {
		t.Errorf("Max should be 3")
	}
}

func TestMin(t *testing.T) {
	if 1 != min(1, 2) {
		t.Errorf("Max should be 1")
	}

	if 2 != min(2, 2) {
		t.Errorf("Max should be 2")
	}

	if 1 != min(3, 1) {
		t.Errorf("Max should be 1")
	}

	if 1 != min(1, 3) {
		t.Errorf("Max should be 1")
	}
}
