package util

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/internal/renderer"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

func RenderCertificateThumbnail(certificate *model.Certificate) error {
	slog.Info("Render Thumbnail starting embedded renderer", "cert_id", certificate.ID)

	// Initialize embedded renderer
	embeddedRenderer, err := renderer.NewEmbeddedRenderer()
	if err != nil {
		slog.Error("Failed to initialize embedded renderer for thumbnail", "error", err, "cert_id", certificate.ID)
		return fmt.Errorf("failed to initialize renderer: %w", err)
	}
	defer embeddedRenderer.Close()

	// Create context with timeout (30 seconds for thumbnail)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if *common.Config.Environment {
		certificate.Design = strings.ReplaceAll(
			certificate.Design,
			"http://easycert.sit.kmutt.ac.th",
			"http://backend:8000",
		)
	}

	// Convert certificate struct to map for renderer compatibility
	certMap := map[string]any{
		"id":     certificate.ID,
		"name":   certificate.Name,
		"design": certificate.Design,
	}

	// Process thumbnail with embedded renderer
	thumbnailPath, err := embeddedRenderer.ProcessThumbnail(ctx, certMap, certificate.ID)
	if err != nil {
		slog.Error("Embedded renderer thumbnail processing failed", "error", err, "cert_id", certificate.ID)
		return fmt.Errorf("thumbnail processing failed: %w", err)
	}

	// Generate presigned URL for thumbnail access
	thumbnailURL := embeddedRenderer.GenerateAccessibleURL(*common.Config.BucketCertificate, thumbnailPath)
	certRepo := certificatemodel.NewCertificateRepository(common.Gorm)
	err = certRepo.AddThumbnailUrl(certificate.ID, thumbnailURL)
	if err != nil {
		slog.Error("Failed to update certificate thumbnail URL", "error", err, "cert_id", certificate.ID, "thumbnail_path", thumbnailPath)
		return fmt.Errorf("failed to update thumbnail URL: %w", err)
	}

	slog.Info("Render Thumbnail successful", "cert_id", certificate.ID, "thumbnail_path", thumbnailPath, "url", thumbnailURL)
	return nil
}

// RenderCertificateThumbnailAsync renders the certificate thumbnail in the background
// This function does not block the calling goroutine and logs any errors that occur
func RenderCertificateThumbnailAsync(certificate *model.Certificate) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic occurred during background thumbnail rendering", "cert_id", certificate.ID, "panic", r)
			}
		}()

		if err := RenderCertificateThumbnail(certificate); err != nil {
			slog.Error("Background thumbnail rendering failed", "error", err, "cert_id", certificate.ID)
		}
	}()

	slog.Info("Background thumbnail rendering started", "cert_id", certificate.ID)
}
