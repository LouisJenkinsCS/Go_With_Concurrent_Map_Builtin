package sync

type KeyType int
type ValueType int

func Interlocked(map[KeyType]ValueType, KeyType, bool, func(*ValueType, bool)) bool
