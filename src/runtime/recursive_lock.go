package runtime

import (
    "runtime/internal/atomic"
    "unsafe"
)

type recursiveLock struct {
    lock mutex
    g *g
}