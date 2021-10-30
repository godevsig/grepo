package efuncs

import (
	"fmt"
	"math"
	"path"
	"runtime"
	"sync/atomic"
	"unsafe"
)

// ErrorHere wraps err with file line info
func ErrorHere(err error) error {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("(%s:%d): %w", path.Base(file), line, err)
}

// AtomicLoadFloat64 loads float64 atomically
func AtomicLoadFloat64(addr *float64) float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(addr))))
}

// AtomicStoreFloat64 stores float64 atomically
func AtomicStoreFloat64(addr *float64, val float64) {
	atomic.StoreUint64((*uint64)(unsafe.Pointer(addr)), math.Float64bits(val))
}
