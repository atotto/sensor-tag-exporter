package influxdb

import (
	"testing"
	"time"
)

func TestTimestamp(t *testing.T) {
	// https://docs.influxdata.com/influxdb/v1.8/write_protocols/line_protocol_tutorial/
	ts := Timestamp(mustTimeParse(t, time.RFC3339Nano, "2016-06-13T17:43:50.1004002Z"))
	expect := "1465839830100400200"
	if ts != expect {
		t.Fatalf("want %s, got %s", expect, ts)
	}
}

func mustTimeParse(tb testing.TB, layout, value string) time.Time {
	tb.Helper()
	t, err := time.Parse(layout, value)
	if err != nil {
		tb.Fatal(err)
	}
	return t
}
