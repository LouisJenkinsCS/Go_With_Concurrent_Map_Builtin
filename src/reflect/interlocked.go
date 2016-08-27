package reflect

import "unsafe"

//go:linkname interlocked sync.Interlocked
func interlocked(map_ interface{}, key interface{}) {
	m := ValueOf(map_)
	k := ValueOf(key)

	m.mustBe(Map)
	tt := (*mapType)(unsafe.Pointer(m.typ))

	// Ensure key is assignable
	k = k.assignTo("reflect.Interlocked", tt.key, nil)

	var k2 unsafe.Pointer
	if (k.flag & flagIndir) != 0 {
		k2 = k.ptr
	} else {
		k2 = unsafe.Pointer(&k.ptr)
	}

	interlockedImpl(m.typ, m.pointer(), k2)
}

func interlockedImpl(maptype *rtype, cmap unsafe.Pointer, key unsafe.Pointer)
