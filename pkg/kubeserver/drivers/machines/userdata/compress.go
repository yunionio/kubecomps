package userdata

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"

	"yunion.io/x/pkg/errors"
)

func CompressUserdata(userdata string) (string, error) {
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(userdata)); err != nil {
		return "", errors.Wrap(err, "failed to gzip userdata")
	}
	if err := gz.Close(); err != nil {
		return "", errors.Wrap(err, "close gzip")
	}
	//return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
	return base64.StdEncoding.EncodeToString([]byte(userdata)), nil
}
