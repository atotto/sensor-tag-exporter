package influxdb

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

func LineProtocol(measurement string, tagSet, fieldSet []string, timestamp time.Time) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := WriteLineProtocol(buf, measurement, tagSet, fieldSet, timestamp); err != nil {
		return nil, fmt.Errorf("write %s", err)
	}
	return buf.Bytes(), nil
}

func WriteLineProtocol(w io.Writer, measurement string, tagSet, fieldSet []string, timestamp time.Time) error {
	// TODO: error
	io.WriteString(w, measurement)
	if tagSet != nil {
		io.WriteString(w, ",")
		stringsJoin(w, tagSet, ",")
	}

	io.WriteString(w, " ")
	stringsJoin(w, fieldSet, ",")

	if !timestamp.IsZero() {
		io.WriteString(w, " ")
		io.WriteString(w, Timestamp(timestamp))
	}
	io.WriteString(w, "\n")

	return nil
}

func PostLine(url string, measurement string, tagSet, fieldSet []string, timestamp time.Time) error {
	body := bytes.NewBuffer(nil)
	if err := WriteLineProtocol(body, measurement, tagSet, fieldSet, timestamp); err != nil {
		return fmt.Errorf("write %s", err)
	}
	return postData(url, body)
}

func PostData(url string, line []byte) error {
	body := bytes.NewReader(line)
	return postData(url, body)
}

func PostBuffer(url string, line io.Reader) error {
	return postData(url, line)
}

func postData(url string, body io.Reader) error {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	return nil
}

func stringsJoin(w io.Writer, elems []string, sep string) {
	switch len(elems) {
	case 0:
		return
	case 1:
		io.WriteString(w, elems[0])
		return
	}
	n := len(sep) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	io.WriteString(w, elems[0])
	for _, s := range elems[1:] {
		io.WriteString(w, sep)
		io.WriteString(w, s)
	}
}
