package influxdb

import (
	"strconv"
	"time"
)

func Timestamp(t time.Time) string {
	return strconv.FormatInt(t.UnixNano(), 10)
}
