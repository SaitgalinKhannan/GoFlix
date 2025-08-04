package httphelpers

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseRange(rawRange string, fileSize int64) (start int64, end int64, err error) {
	if rawRange == "" {
		return 0, fileSize - 1, nil // Запрос всего файла
	}

	const prefix = "bytes="
	if !strings.HasPrefix(rawRange, prefix) {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	rangeStr := strings.TrimPrefix(rawRange, prefix)
	parts := strings.Split(rangeStr, "-")

	// Парсим start
	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return 0, 0, fmt.Errorf("invalid start range")
	}

	// Парсим end
	if len(parts) < 2 || parts[1] == "" {
		end = fileSize - 1 // bytes=500- → до конца файла
	} else {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return 0, 0, fmt.Errorf("invalid end range")
		}
	}

	// Корректируем end, если он больше размера файла
	if end >= fileSize {
		end = fileSize - 1
	}

	return start, end, nil
}
