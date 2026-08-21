package sys

import (
	"strconv"
	"strings"

	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/errors/gerror"
)

// parseByteRange 解析 RFC 7233 单段 Range。header 为空表示整文件。
func parseByteRange(header string, size int64) (start, end int64, partial bool, err error) {
	h := strings.TrimSpace(header)
	if h == "" {
		if size <= 0 {
			return 0, -1, false, nil
		}
		return 0, size - 1, false, nil
	}
	if size <= 0 {
		return 0, 0, true, &sysin.UnsatisfiableRangeError{Size: 0}
	}
	if !strings.HasPrefix(strings.ToLower(h), "bytes=") {
		return 0, 0, false, gerror.New("不支持的 Range 格式")
	}
	spec := strings.TrimSpace(h[len("bytes="):])
	if i := strings.Index(spec, ","); i >= 0 {
		spec = strings.TrimSpace(spec[:i])
	}
	dash := strings.Index(spec, "-")
	if dash < 0 {
		return 0, 0, false, gerror.New("不支持的 Range 格式")
	}
	from := strings.TrimSpace(spec[:dash])
	to := strings.TrimSpace(spec[dash+1:])

	if from == "" {
		n, convErr := strconv.ParseInt(to, 10, 64)
		if convErr != nil || n <= 0 {
			return 0, 0, false, gerror.New("不支持的 Range 格式")
		}
		if n > size {
			n = size
		}
		return size - n, size - 1, true, nil
	}

	start, convErr := strconv.ParseInt(from, 10, 64)
	if convErr != nil || start < 0 {
		return 0, 0, false, gerror.New("不支持的 Range 格式")
	}
	if start >= size {
		return 0, 0, true, &sysin.UnsatisfiableRangeError{Size: size}
	}
	if to == "" {
		return start, size - 1, true, nil
	}
	end, convErr = strconv.ParseInt(to, 10, 64)
	if convErr != nil || end < start {
		return 0, 0, false, gerror.New("不支持的 Range 格式")
	}
	if end >= size {
		end = size - 1
	}
	return start, end, true, nil
}
