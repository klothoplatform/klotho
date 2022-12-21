package analytics

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/klothoplatform/klotho/pkg/cli"
	"github.com/klothoplatform/klotho/pkg/core"
)

var kloServerUrl = "http://srv.klo.dev"

type AnalyticsFile struct {
	Email string
	Id    string
}

func SendTrackingToServer(bundle *Client) error {
	postBody, _ := json.Marshal(bundle)
	data := bytes.NewBuffer(postBody)
	resp, err := http.Post(fmt.Sprintf("%v/analytics/track", kloServerUrl), "application/json", data)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
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

func getTrackingFileContents(file string) (AnalyticsFile, error) {
	configPath, err := cli.KlothoConfigPath(file)
	result := AnalyticsFile{}

	if err != nil {
		return result, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(content, &result)

	return result, err
}
