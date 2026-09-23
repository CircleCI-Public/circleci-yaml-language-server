// Package tsalloc counts tree-sitter's live allocations, so that a test can
// tell whether the code it ran closed every tree, parser, query and cursor it
// made.
//
// The official Go bindings free C memory only when an object is closed, so a
// forgotten Close is a leak that nothing else reports. tree-sitter routes all
// of its allocations through an allocator that can be replaced; this one
// counts them on their way to the C allocator.
package tsalloc

/*
#include <stdlib.h>
*/
import "C"

import (
	"sync"
	"testing"
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

var (
	// allocated holds every pointer the counting allocator handed out and
	// has not yet been asked to free. It is a set rather than a count because
	// the bindings allocate some strings with C's malloc directly and free
	// them through tree-sitter's free: those frees must be ignored, not
	// subtracted.
	mutex     sync.Mutex
	allocated = map[unsafe.Pointer]struct{}{}
	install   sync.Once
)

func record(ptr unsafe.Pointer) unsafe.Pointer {
	if ptr != nil {
		mutex.Lock()
		allocated[ptr] = struct{}{}
		mutex.Unlock()
	}
	return ptr
}

func forget(ptr unsafe.Pointer) {
	mutex.Lock()
	delete(allocated, ptr)
	mutex.Unlock()
}

func malloc(size uint) unsafe.Pointer {
	return record(C.malloc(C.size_t(size)))
}

func calloc(num, size uint) unsafe.Pointer {
	return record(C.calloc(C.size_t(num), C.size_t(size)))
}

func realloc(ptr unsafe.Pointer, size uint) unsafe.Pointer {
	mutex.Lock()
	_, ours := allocated[ptr]
	mutex.Unlock()

	moved := C.realloc(ptr, C.size_t(size))
	if moved == nil && size != 0 {
		// A failed realloc leaves the original where it was.
		return nil
	}
	if ptr != nil {
		forget(ptr)
	}
	if ptr == nil || ours {
		record(moved)
	}
	return moved
}

func free(ptr unsafe.Pointer) {
	if ptr != nil {
		forget(ptr)
	}
	C.free(ptr)
}

// Live reports how many of tree-sitter's allocations since the counting
// allocator was installed have not been freed.
func Live() int64 {
	mutex.Lock()
	defer mutex.Unlock()

	return int64(len(allocated))
}

// Track installs the counting allocator, once per test binary, and returns a
// function reporting how many allocations made since Track was called are
// still live. Tests that use it must not run in parallel with other code that
// parses, or they count each other's allocations.
func Track(t testing.TB) func() int64 {
	t.Helper()

	install.Do(func() {
		sitter.SetAllocator(malloc, calloc, realloc, free)
	})

	before := Live()

	return func() int64 {
		return Live() - before
	}
}
