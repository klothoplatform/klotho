package analytics

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"math"
	"net/http"
	"time"

	"github.com/klothoplatform/klotho/pkg/core"
)

const kloServerUrl = "http://srv.klo.dev"

type AnalyticsFile struct {
	Id string
}

func (t *Client) Send(payload *Payload) {
	postBody, _ := json.Marshal(payload)
	data := bytes.NewBuffer(postBody)
	url := t.serverUrlOverride
	if url == "" {
		url = kloServerUrl
	}
	resp, err := http.Post(fmt.Sprintf("%v/analytics/track", url), "application/json", data)

	if err != nil {
		zap.L().Debug(fmt.Sprintf("Failed to send metrics info. %v", err))
		return
	}
	resp.Body.Close()
}

func CompressFiles(input *core.InputFiles) ([]byte, error) {
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)
	now := time.Now().UTC()

	for _, f := range input.Files() {
		buf := new(bytes.Buffer)
		if _, err := f.WriteTo(buf); err != nil {
			return nil, err
		}

		header := &zip.FileHeader{
			Method:             zip.Deflate,
			Name:               f.Path(),
			UncompressedSize64: uint64(buf.Len()),
			Modified:           now,
		}
		if header.UncompressedSize64 >= math.MaxUint32 {
			header.UncompressedSize = math.MaxUint32
		} else {
			header.UncompressedSize = uint32(header.UncompressedSize64)
		}

		headerWriter, err := zipWriter.CreateHeader(header)
		if err != nil {
			return nil, err
		}

		if _, err := buf.WriteTo(headerWriter); err != nil {
			return nil, err
		}
	}

	err := zipWriter.Close()

	return buf.Bytes(), err
}
