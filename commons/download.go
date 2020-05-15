package commons

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func DownloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return errors.New(fmt.Sprintf("download failed with status code %s", resp.StatusCode))
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}
