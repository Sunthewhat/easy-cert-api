package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

func RenderCertificateThumbnail(certificate *model.Certificate) error {
	requestBody := map[string]any{
		"certificate": certificate,
	}

	jsonData, marshalErr := json.Marshal(requestBody)
	if marshalErr != nil {
		slog.Error("Render Thumbnail Marshal Failed", "cert", certificate, "error", marshalErr)
		return marshalErr
	}

	rendererUrl := fmt.Sprintf("%s/api/thumbnail", *common.Config.RendererUrl)

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	req, reqErr := http.NewRequest("POST", rendererUrl, bytes.NewBuffer(jsonData))
	if reqErr != nil {
		slog.Error("Render Thumbnail request creation failed", "error", reqErr)
		return reqErr
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, respErr := client.Do(req)
	if respErr != nil {
		slog.Error("Render Thumbnail HTTP request failed", "error", respErr)
		return respErr
	}

	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		slog.Error("Render Thumbnail response read failed", "error", readErr)
		return readErr
	}

	if resp.StatusCode == 200 {
		slog.Info("Render Thumbnail Sucessful")

		var renderResponse payload.RenderThumbnailPayload
		if parseErr := json.Unmarshal(responseBody, &renderResponse); parseErr == nil {
			err := certificatemodel.AddThumbnailUrl(certificate.ID, fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, *common.Config.BucketCertificate, renderResponse.ThumbnailPath))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
