package test

import (
	"testing"
	"u"
)

func TestShortUUID(t *testing.T) {
	for i := 1; i <= 32; i++ {
		uid := u.ShortUUID(i)
		t.Logf("length=%d, uuid=%v\n", i, uid)
		if len(uid) != i {
			t.Errorf("wanted length %d, got length %d, uuid=%v\n", i, len(uid), uid)
		}
	}
}

func TestIsValueNilBenchmark(t *testing.T) {
	var ptr *int
	var itf interface{}
	itf = ptr
	if !u.IsValueNil(itf) {
		t.Errorf("expect u.IsValueNil(itf) to return true. itf=%#v", itf)
	}
}

func BenchmarkIsValueNil(b *testing.B) {
	var ptr *int
	var itf interface{}
	itf = ptr
	for i := 0; i < b.N; i++ {
		u.IsValueNil(itf)
	}
}

func BenchmarkIsNil(b *testing.B) {
	var ptr *int
	var itf interface{}
	itf = ptr
	for i := 0; i < b.N; i++ {
		isNil(itf)
	}
}

func isNil(i interface{}) bool {
	return i == nil
}
