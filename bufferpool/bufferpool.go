package bufferpool

import (
	"sync"
)

const bufferSize = 1024

var bytePool = sync.Pool{
	New: func() any {
		return make([]byte, bufferSize)
	},
}

func GetBytes() []byte {
	return bytePool.Get().([]byte)
}

func PutBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	bytePool.Put(b)
}
