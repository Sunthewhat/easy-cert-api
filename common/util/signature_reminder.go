package util

import (
	"log/slog"
	"time"

	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
	"github.com/sunthewhat/easy-cert-api/common"
)

// StartSignatureReminderJob starts a background job that sends daily reminder emails
// to signers who haven't signed their certificates yet
func StartSignatureReminderJob() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic occurred in signature reminder job", "panic", r)
			}
		}()

		// Run immediately on startup to catch any pending reminders
		slog.Info("Signature reminder job: Initial run starting")
		SendSignatureReminders()

		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			slog.Info("Signature reminder job: Scheduled run starting")
			SendSignatureReminders()
		}
	}()

	slog.Info("Signature reminder job started successfully")
}

// SendSignatureReminders finds all pending signatures and sends reminder emails
func SendSignatureReminders() {
	startTime := time.Now()
	slog.Info("SendSignatureReminders: Starting reminder process")

	// Get pending signatures
	pendingSignatures, err := signaturemodel.GetPendingSignaturesForReminder()
	if err != nil {
		slog.Error("SendSignatureReminders: Failed to get pending signatures", "error", err)
		return
	}

	if len(pendingSignatures) == 0 {
		duration := time.Since(startTime)
		slog.Info("SendSignatureReminders: No pending signatures", "duration", duration)
		return
	}

	var successCount, failedCount int

	for _, signature := range pendingSignatures {
		// Get signer details
		signer, err := signermodel.GetById(signature.SignerID)
		if err != nil {
			slog.Error("SendSignatureReminders: Error getting signer", "error", err, "signerId", signature.SignerID)
			failedCount++
			continue
		}

		if signer == nil {
			slog.Warn("SendSignatureReminders: Signer not found", "signerId", signature.SignerID)
			failedCount++
			continue
		}

		// Get certificate details
		certRepo := certificatemodel.NewCertificateRepository(common.Gorm)
		certificate, err := certRepo.GetById(signature.CertificateID)
		if err != nil {
			slog.Error("SendSignatureReminders: Error getting certificate", "error", err, "certificateId", signature.CertificateID)
			failedCount++
			continue
		}

		if certificate == nil {
			slog.Warn("SendSignatureReminders: Certificate not found", "certificateId", signature.CertificateID)
			failedCount++
			continue
		}

		// Send reminder email
		err = SendSignatureReminderMail(signer.Email, signer.DisplayName, certificate.ID, certificate.Name)
		if err != nil {
			slog.Error("SendSignatureReminders: Failed to send reminder", "error", err, "signerId", signature.SignerID)
			failedCount++
			continue
		}

		// Update last_request timestamp
		markErr := signaturemodel.MarkAsRequested(certificate.ID, signature.SignerID)
		if markErr != nil {
			slog.Warn("SendSignatureReminders: Failed to update last_request", "error", markErr, "signerId", signature.SignerID)
		}

		successCount++
	}

	duration := time.Since(startTime)
	slog.Info("SendSignatureReminders: Completed", "total", len(pendingSignatures), "success", successCount, "failed", failedCount, "duration", duration)
}
