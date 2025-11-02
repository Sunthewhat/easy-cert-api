package util

import (
	"log/slog"
	"time"

	"github.com/sunthewhat/easy-cert-api/internal/renderer"
)

// StartPreviewCleanupJob starts a background job that cleans up old preview images
// Preview images older than 30 days will be automatically deleted
func StartPreviewCleanupJob() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic occurred in preview cleanup job", "panic", r)
			}
		}()

		// Run immediately on startup to clean up any old previews
		slog.Info("Preview cleanup job: Initial run starting")
		CleanupOldPreviews()

		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			slog.Info("Preview cleanup job: Scheduled run starting")
			CleanupOldPreviews()
		}
	}()

	slog.Info("Preview cleanup job started successfully")
}

// CleanupOldPreviews removes preview files older than 30 days
func CleanupOldPreviews() {
	startTime := time.Now()
	slog.Info("CleanupOldPreviews: Starting cleanup process")

	// Initialize embedded renderer for cleanup
	embeddedRenderer, err := renderer.NewEmbeddedRenderer()
	if err != nil {
		slog.Error("CleanupOldPreviews: Failed to initialize renderer", "error", err)
		return
	}
	defer embeddedRenderer.Close()

	// Clean up previews older than 30 days
	maxAge := 30 * 24 * time.Hour
	err = embeddedRenderer.CleanupExpiredPreviews(maxAge)
	if err != nil {
		slog.Error("CleanupOldPreviews: Cleanup failed", "error", err, "duration", time.Since(startTime))
		return
	}

	duration := time.Since(startTime)
	slog.Info("CleanupOldPreviews: Completed successfully", "maxAge", maxAge.String(), "duration", duration)
}
