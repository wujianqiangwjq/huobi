package huobi

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"time"
)

func GetUinxMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
func UnGzip(buf []byte) ([]byte, error) {
	r, er := gzip.NewReader(bytes.NewBuffer(buf))
	if er != nil {
		return nil, er
	}
	return ioutil.ReadAll(r)
}
