package helper

import (
	"io"
)

func Copy(dst io.Writer, src io.Reader) (int64, error) {
	var err error
	var written int64 = 0
	buff := make([]byte, 32*1024)
	for {
		n, err := src.Read(buff)
		if err != nil {
			break
		}
		_, err = dst.Write(buff[:n])
		if err != nil {
			break
		}
		written += int64(n)
	}
	return written, err
}
