package util

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"gopkg.in/gomail.v2"
)

func InitDialer() {
	dailer := gomail.NewDialer(*common.Config.MailHost, 587, *common.Config.MailUser, *common.Config.MailPass)
	common.Dialer = dailer
}

func SendMail(participantMail string, certificateUrl string) error {
	// Generate unique filename to avoid conflicts
	uniqueID := uuid.New().String()
	timestamp := time.Now().Unix()
	fileUrl := fmt.Sprintf("Certificate_%s_%d.pdf", uniqueID, timestamp)

	if err := downloadCertificate(certificateUrl, fileUrl); err != nil {
		slog.Error("Sendmail Util Error Downloading File", "error", err)
		return err
	}

	// Check if file was downloaded correctly
	if err := validateDownloadedFile(fileUrl); err != nil {
		slog.Error("Downloaded file validation failed", "error", err)
		os.Remove(fileUrl)
		return err
	}

	mailer := gomail.NewMessage()
	mailer.SetHeader("From", *common.Config.MailUser)
	mailer.SetHeader("To", participantMail)
	mailer.SetHeader("Subject", "Your Certificate")
	mailer.SetBody("text/html", `
		<p>Dear Participant,</p>
		<p>Please find your certificate attached to this email.</p>
		<p>Best regards,<br>Easy Cert Team</p>
	`)

	// Attach with proper filename and content type
	mailer.Attach(fileUrl, gomail.Rename("Certificate.pdf"), gomail.SetHeader(map[string][]string{
		"Content-Type": {"application/pdf"},
	}))

	if err := common.Dialer.DialAndSend(mailer); err != nil {
		slog.Error("Error Sending Mail", "error", err)
		os.Remove(fileUrl)
		return err
	}

	os.Remove(fileUrl)
	slog.Info("Email sent successfully", "recipient", participantMail)

	return nil
}

func downloadCertificate(url string, filename string) error {
	if *common.Config.Environment {
		url = strings.ReplaceAll(
			url,
			"http://easycert.sit.kmutt.ac.th",
			"http://backend:8000",
		)
	}
	slog.Info("Downloading certificate", "url", url, "filename", filename)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Check Content-Type (optional but recommended)
	contentType := resp.Header.Get("Content-Type")
	slog.Info("Downloaded file info", "content-type", contentType, "content-length", resp.ContentLength)

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy the response body to file
	bytesWritten, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	slog.Info("File downloaded successfully", "bytes", bytesWritten)
	return nil
}

func validateDownloadedFile(filename string) error {
	stat, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Check if file is empty
	if stat.Size() == 0 {
		return fmt.Errorf("downloaded file is empty")
	}

	// For PDF files, check if it starts with PDF header
	if filepath.Ext(filename) == ".pdf" {
		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("cannot open file for validation: %w", err)
		}
		defer file.Close()

		header := make([]byte, 4)
		_, err = file.Read(header)
		if err != nil {
			return fmt.Errorf("cannot read file header: %w", err)
		}

		if string(header) != "%PDF" {
			return fmt.Errorf("file is not a valid PDF (header: %s)", string(header))
		}
	}

	slog.Info("File validation passed", "filename", filename, "size", stat.Size())
	return nil
}

// SendSignatureRequestMail sends an email to a signer requesting them to sign a certificate
func SendSignatureRequestMail(signerEmail, signerName, certificateId, certificateName string) error {
	signatureURL := fmt.Sprintf("%s/signature/%s", *common.Config.VerifyHost, certificateId)

	mailer := gomail.NewMessage()
	mailer.SetHeader("From", *common.Config.MailUser)
	mailer.SetHeader("To", signerEmail)
	mailer.SetHeader("Subject", fmt.Sprintf("Signature Request - %s", certificateName))

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<link href="https://fonts.googleapis.com/css2?family=Noto+Sans+Thai:wght@400;600;700&display=swap" rel="stylesheet">
			<style>
				body {
					font-family: 'Noto Sans Thai', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
					line-height: 1.6;
					margin: 0;
					padding: 0;
					background: linear-gradient(135deg, #e5e7eb 0%%, #d5d8de 100%%);
				}
				.container {
					max-width: 600px;
					margin: 40px auto;
					background: rgba(255, 255, 255, 0.95);
					border-radius: 28px;
					border: 1px solid rgba(255, 255, 255, 0.6);
					box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
					overflow: hidden;
				}
				.header {
					background: linear-gradient(135deg, #244dad 0%%, #1e3d8f 100%%);
					color: white;
					padding: 48px 32px;
					text-align: center;
				}
				.header h1 {
					margin: 0;
					font-size: 28px;
					font-weight: 700;
					letter-spacing: -0.02em;
				}
				.header p {
					margin: 8px 0 0 0;
					font-size: 15px;
					opacity: 0.9;
				}
				.content {
					padding: 40px 32px;
				}
				.greeting {
					font-size: 18px;
					font-weight: 600;
					color: #1f2937;
					margin-bottom: 16px;
				}
				.message {
					font-size: 16px;
					color: #374151;
					margin-bottom: 24px;
					line-height: 1.7;
				}
				.cert-card {
					background: linear-gradient(135deg, rgba(229, 231, 235, 0.4) 0%%, rgba(255, 255, 255, 0.6) 100%%);
					border: 1px solid rgba(36, 77, 173, 0.15);
					border-radius: 20px;
					padding: 24px;
					margin: 28px 0;
				}
				.cert-name {
					font-size: 20px;
					font-weight: 700;
					color: #244dad;
					margin: 0;
				}
				.button {
					display: inline-block;
					background: #244dad;
					color: white;
					padding: 14px 32px;
					border-radius: 100px;
					text-decoration: none;
					font-weight: 600;
					font-size: 15px;
					margin: 24px 0;
					box-shadow: 0 10px 25px -5px rgba(36, 77, 173, 0.3);
				}
				.link-text {
					font-size: 13px;
					color: #6b7280;
					word-break: break-all;
					background: rgba(229, 231, 235, 0.5);
					padding: 12px 16px;
					border-radius: 8px;
					margin: 16px 0;
				}
				.footer {
					background: rgba(249, 250, 251, 0.8);
					padding: 32px;
					text-align: center;
					font-size: 13px;
					color: #9ca3af;
					border-top: 1px solid rgba(229, 231, 235, 0.8);
				}
				.footer p {
					margin: 8px 0;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Signature Request</h1>
					<p>Your signature is needed</p>
				</div>
				<div class="content">
					<p class="greeting">Dear %s,</p>
					<p class="message">
						You have been requested to sign the following certificate. Your signature is an important part of this verification process.
					</p>
					<div class="cert-card">
						<p class="cert-name">%s</p>
					</div>
					<p class="message">
						Please click the button below to review and sign the certificate:
					</p>
					<center>
						<a href="%s" class="button">Sign Certificate →</a>
					</center>
					<p style="font-size: 14px; color: #6b7280; text-align: center; margin-top: 16px;">Or copy this link to your browser:</p>
					<div class="link-text">%s</div>
				</div>
				<div class="footer">
					<p><strong>EasyCert</strong> - Secure Certificate Management</p>
					<p style="margin-top: 12px;">If you did not expect this email, please ignore it.</p>
				</div>
			</div>
		</body>
		</html>
	`, signerName, certificateName, signatureURL, signatureURL)

	mailer.SetBody("text/html", htmlBody)

	if err := common.Dialer.DialAndSend(mailer); err != nil {
		slog.Error("Error sending signature request email", "error", err, "recipient", signerEmail, "certificateId", certificateId)
		return err
	}

	slog.Info("Signature request email sent successfully", "recipient", signerEmail, "certificateId", certificateId)
	return nil
}

// SendSignatureReminderMail sends a reminder email to a signer
func SendSignatureReminderMail(signerEmail, signerName, certificateId, certificateName string) error {
	signatureURL := fmt.Sprintf("%s/signature/%s", *common.Config.VerifyHost, certificateId)

	mailer := gomail.NewMessage()
	mailer.SetHeader("From", *common.Config.MailUser)
	mailer.SetHeader("To", signerEmail)
	mailer.SetHeader("Subject", fmt.Sprintf("Reminder: Signature Request - %s", certificateName))

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<link href="https://fonts.googleapis.com/css2?family=Noto+Sans+Thai:wght@400;600;700&display=swap" rel="stylesheet">
			<style>
				body {
					font-family: 'Noto Sans Thai', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
					line-height: 1.6;
					margin: 0;
					padding: 0;
					background: linear-gradient(135deg, #e5e7eb 0%%, #d5d8de 100%%);
				}
				.container {
					max-width: 600px;
					margin: 40px auto;
					background: rgba(255, 255, 255, 0.95);
					border-radius: 28px;
					border: 1px solid rgba(255, 255, 255, 0.6);
					box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
					overflow: hidden;
				}
				.header {
					background: linear-gradient(135deg, #f59e0b 0%%, #d97706 100%%);
					color: white;
					padding: 48px 32px;
					text-align: center;
				}
				.header h1 {
					margin: 0;
					font-size: 28px;
					font-weight: 700;
					letter-spacing: -0.02em;
				}
				.header p {
					margin: 8px 0 0 0;
					font-size: 15px;
					opacity: 0.9;
				}
				.content {
					padding: 40px 32px;
				}
				.reminder-badge {
					background: linear-gradient(135deg, #fef3c7 0%%, #fde68a 100%%);
					color: #92400e;
					display: inline-block;
					padding: 10px 20px;
					border-radius: 100px;
					font-size: 14px;
					font-weight: 600;
					margin-bottom: 24px;
				}
				.greeting {
					font-size: 18px;
					font-weight: 600;
					color: #1f2937;
					margin-bottom: 16px;
				}
				.message {
					font-size: 16px;
					color: #374151;
					margin-bottom: 24px;
					line-height: 1.7;
				}
				.cert-card {
					background: linear-gradient(135deg, rgba(254, 243, 199, 0.3) 0%%, rgba(253, 230, 138, 0.2) 100%%);
					border: 1px solid rgba(245, 158, 11, 0.2);
					border-radius: 20px;
					padding: 24px;
					margin: 28px 0;
				}
				.cert-name {
					font-size: 20px;
					font-weight: 700;
					color: #d97706;
					margin: 0;
				}
				.button {
					display: inline-block;
					background: #f59e0b;
					color: white;
					padding: 14px 32px;
					border-radius: 100px;
					text-decoration: none;
					font-weight: 600;
					font-size: 15px;
					margin: 24px 0;
					box-shadow: 0 10px 25px -5px rgba(245, 158, 11, 0.4);
				}
				.link-text {
					font-size: 13px;
					color: #6b7280;
					word-break: break-all;
					background: rgba(229, 231, 235, 0.5);
					padding: 12px 16px;
					border-radius: 8px;
					margin: 16px 0;
				}
				.footer {
					background: rgba(249, 250, 251, 0.8);
					padding: 32px;
					text-align: center;
					font-size: 13px;
					color: #9ca3af;
					border-top: 1px solid rgba(229, 231, 235, 0.8);
				}
				.footer p {
					margin: 8px 0;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Signature Reminder</h1>
					<p>Your signature is still needed</p>
				</div>
				<div class="content">
					<div class="reminder-badge">PENDING</div>
					<p class="greeting">Dear %s,</p>
					<p class="message">
						This is a friendly reminder that you have a pending signature request for the following certificate. Your signature is important for completing this verification process.
					</p>
					<div class="cert-card">
						<p class="cert-name">%s</p>
					</div>
					<p class="message">
						Please take a moment to review and sign the certificate:
					</p>
					<center>
						<a href="%s" class="button">Sign Certificate Now →</a>
					</center>
					<p style="font-size: 14px; color: #6b7280; text-align: center; margin-top: 16px;">Or copy this link to your browser:</p>
					<div class="link-text">%s</div>
				</div>
				<div class="footer">
					<p><strong>EasyCert</strong> - Secure Certificate Management</p>
					<p style="margin-top: 12px;">You will receive reminders until the certificate is signed. If you did not expect this email, please ignore it.</p>
				</div>
			</div>
		</body>
		</html>
	`, signerName, certificateName, signatureURL, signatureURL)

	mailer.SetBody("text/html", htmlBody)

	if err := common.Dialer.DialAndSend(mailer); err != nil {
		slog.Error("Error sending signature reminder email", "error", err, "recipient", signerEmail, "certificateId", certificateId)
		return err
	}

	slog.Info("Signature reminder email sent successfully", "recipient", signerEmail, "certificateId", certificateId)
	return nil
}

// BulkSendSignatureRequests sends signature request emails to multiple signers
func BulkSendSignatureRequests(certificateId, certificateName string, signerIds []string) error {
	if len(signerIds) == 0 {
		return nil
	}

	var successCount, failedCount int
	var lastError error

	signerRepo := signermodel.NewSignerRepository(common.Gorm)
	signatureRepo := signaturemodel.NewSignatureRepository(common.Gorm)

	for _, signerId := range signerIds {
		// Get signer details
		signer, err := signerRepo.GetById(signerId)
		if err != nil {
			slog.Error("BulkSendSignatureRequests: Error getting signer", "error", err, "signerId", signerId, "certificateId", certificateId)
			failedCount++
			lastError = err
			continue
		}

		if signer == nil {
			slog.Warn("BulkSendSignatureRequests: Signer not found", "signerId", signerId, "certificateId", certificateId)
			failedCount++
			lastError = fmt.Errorf("signer %s not found", signerId)
			continue
		}

		// Send signature request email
		err = SendSignatureRequestMail(signer.Email, signer.DisplayName, certificateId, certificateName)
		if err != nil {
			slog.Error("BulkSendSignatureRequests: Failed to send email", "error", err, "signerId", signerId, "email", signer.Email, "certificateId", certificateId)
			failedCount++
			lastError = err
			continue
		}

		// Mark signature as requested after successful email send
		markErr := signatureRepo.MarkAsRequested(certificateId, signerId)
		if markErr != nil {
			slog.Warn("BulkSendSignatureRequests: Failed to mark as requested", "error", markErr, "signerId", signerId, "certificateId", certificateId)
			// Don't fail if marking fails - email was sent successfully
		}

		successCount++
	}

	slog.Info("BulkSendSignatureRequests: Completed", "certificateId", certificateId, "total", len(signerIds), "success", successCount, "failed", failedCount)

	// Only return error if all emails failed
	if failedCount > 0 && successCount == 0 {
		return fmt.Errorf("failed to send all signature request emails: %w", lastError)
	}

	return nil
}

// SendAllSignaturesCompleteMail sends notification to certificate owner when all signatures are complete
// with an optional preview image attachment
func SendAllSignaturesCompleteMail(ownerEmail, certificateName, certificateId, previewPath string) error {
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", *common.Config.MailUser)
	mailer.SetHeader("To", ownerEmail)
	mailer.SetHeader("Subject", fmt.Sprintf("All Signatures Complete - %s", certificateName))

	// Build HTML body with preview mention if preview is available
	previewSection := ""
	if previewPath != "" {
		previewSection = `
					<div style="background: linear-gradient(135deg, rgba(36, 77, 173, 0.05) 0%, rgba(36, 77, 173, 0.02) 100%); border: 2px solid rgba(36, 77, 173, 0.1); border-radius: 16px; padding: 24px; margin: 24px 0; text-align: center;">
						<p style="margin: 0 0 12px 0; font-size: 15px; color: #244dad; font-weight: 600;">Preview Attached</p>
						<p style="margin: 0; font-size: 14px; color: #6b7280; line-height: 1.6;">A preview of the signed certificate is attached to this email. Note: The preview includes a watermark and is for reference only.</p>
					</div>`
	}

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<link href="https://fonts.googleapis.com/css2?family=Noto+Sans+Thai:wght@400;600;700&display=swap" rel="stylesheet">
			<style>
				body {
					font-family: 'Noto Sans Thai', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
					line-height: 1.6;
					margin: 0;
					padding: 0;
					background: linear-gradient(135deg, #e5e7eb 0%%, #d5d8de 100%%);
				}
				.container {
					max-width: 600px;
					margin: 40px auto;
					background: rgba(255, 255, 255, 0.95);
					border-radius: 28px;
					border: 1px solid rgba(255, 255, 255, 0.6);
					box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
					overflow: hidden;
				}
				.header {
					background: linear-gradient(135deg, #244dad 0%%, #1e3d8f 100%%);
					color: white;
					padding: 48px 32px;
					text-align: center;
				}
				.header h1 {
					margin: 0;
					font-size: 28px;
					font-weight: 700;
					letter-spacing: -0.02em;
				}
				.header p {
					margin: 8px 0 0 0;
					font-size: 15px;
					opacity: 0.9;
				}
				.content {
					padding: 40px 32px;
				}
				.success-badge {
					background: linear-gradient(135deg, #10b981 0%%, #059669 100%%);
					color: white;
					display: inline-block;
					padding: 10px 20px;
					border-radius: 100px;
					font-size: 14px;
					font-weight: 600;
					margin-bottom: 24px;
				}
				.message {
					font-size: 16px;
					color: #374151;
					margin-bottom: 24px;
					line-height: 1.7;
				}
				.cert-card {
					background: linear-gradient(135deg, rgba(229, 231, 235, 0.4) 0%%, rgba(255, 255, 255, 0.6) 100%%);
					border: 1px solid rgba(36, 77, 173, 0.15);
					border-radius: 20px;
					padding: 24px;
					margin: 28px 0;
				}
				.cert-name {
					font-size: 20px;
					font-weight: 700;
					color: #244dad;
					margin: 0 0 8px 0;
				}
				.cert-id {
					font-size: 13px;
					color: #6b7280;
					font-family: 'Courier New', monospace;
					margin: 0;
				}
				.button {
					display: inline-block;
					background: #244dad;
					color: white;
					padding: 14px 32px;
					border-radius: 100px;
					text-decoration: none;
					font-weight: 600;
					font-size: 15px;
					margin: 24px 0;
					box-shadow: 0 10px 25px -5px rgba(36, 77, 173, 0.3);
				}
				.footer {
					background: rgba(249, 250, 251, 0.8);
					padding: 32px;
					text-align: center;
					font-size: 13px;
					color: #9ca3af;
					border-top: 1px solid rgba(229, 231, 235, 0.8);
				}
				.footer p {
					margin: 8px 0;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>All Signatures Complete</h1>
					<p>Your certificate is ready</p>
				</div>
				<div class="content">
					<div class="success-badge">Signing Complete</div>
					<p class="message">
						Great news! All required signatures have been successfully collected for your certificate.
						The signing process is now complete.
					</p>
					<div class="cert-card">
						<p class="cert-name">%s</p>
						<p class="cert-id">ID: %s</p>
					</div>
					%s
					<p class="message">
						You can now generate and distribute the fully signed certificate through your EasyCert dashboard.
					</p>
					<center>
						<a href="%s" class="button">View Dashboard →</a>
					</center>
				</div>
				<div class="footer">
					<p><strong>EasyCert</strong> - Secure Certificate Management</p>
					<p style="margin-top: 12px;">This is an automated notification. Please do not reply to this email.</p>
				</div>
			</div>
		</body>
		</html>
	`, certificateName, certificateId, previewSection, *common.Config.VerifyHost)

	mailer.SetBody("text/html", htmlBody)

	// Attach preview image if available
	if previewPath != "" {
		// Download preview from MinIO
		previewFile, downloadErr := downloadPreviewFromMinIO(previewPath)
		if downloadErr != nil {
			slog.Warn("Failed to download preview for email attachment", "error", downloadErr, "previewPath", previewPath)
			// Continue sending email without preview
		} else {
			defer os.Remove(previewFile) // Clean up temp file after sending

			// Attach preview image
			mailer.Attach(previewFile, gomail.Rename("certificate_preview.png"), gomail.SetHeader(map[string][]string{
				"Content-Type": {"image/png"},
			}))
			slog.Info("Preview attached to email", "previewPath", previewPath, "recipient", ownerEmail)
		}
	}

	if err := common.Dialer.DialAndSend(mailer); err != nil {
		slog.Error("Failed to send all signatures complete email", "error", err, "recipient", ownerEmail, "certificateId", certificateId)
		return err
	}

	slog.Info("All signatures complete email sent successfully", "recipient", ownerEmail, "certificateId", certificateId, "withPreview", previewPath != "")
	return nil
}

// downloadPreviewFromMinIO downloads a preview image from MinIO to a temporary file
func downloadPreviewFromMinIO(objectPath string) (string, error) {
	bucketName := *common.Config.BucketCertificate

	// Create temporary file
	tempFile, err := os.CreateTemp("", "preview-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Download from MinIO
	ctx := context.Background()
	object, err := common.MinIOClient.GetObject(ctx, bucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	defer object.Close()

	// Copy to temp file
	_, err = io.Copy(tempFile, object)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to copy preview to temp file: %w", err)
	}

	slog.Info("Preview downloaded from MinIO", "objectPath", objectPath, "tempFile", tempFile.Name())
	return tempFile.Name(), nil
}
