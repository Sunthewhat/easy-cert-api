package util

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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
	mailer.SetHeader("Subject", fmt.Sprintf("Signature Request for Certificate: %s", certificateName))

	// HTML email body with better formatting
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #3b82f6; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
				.content { background-color: #f9fafb; padding: 30px; border: 1px solid #e5e7eb; }
				.certificate-name { font-weight: bold; color: #1f2937; font-size: 18px; margin: 15px 0; }
				.button { display: inline-block; padding: 12px 30px; background-color: #3b82f6; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
				.button:hover { background-color: #2563eb; }
				.footer { color: #6b7280; font-size: 14px; margin-top: 20px; padding-top: 20px; border-top: 1px solid #e5e7eb; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h2>Signature Request</h2>
				</div>
				<div class="content">
					<p>Dear %s,</p>
					<p>You have been requested to sign the following certificate:</p>
					<div class="certificate-name">"%s"</div>
					<p>Please click the button below to review and sign the certificate:</p>
					<a href="%s" class="button">Sign Certificate</a>
					<p>Or copy this link to your browser:</p>
					<p style="word-break: break-all; color: #3b82f6;">%s</p>
					<div class="footer">
						<p>Best regards,<br>Easy Cert Team</p>
						<p style="font-size: 12px; color: #9ca3af;">If you did not expect this email, please ignore it.</p>
					</div>
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
	mailer.SetHeader("Subject", fmt.Sprintf("Reminder: Signature Request for Certificate: %s", certificateName))

	// HTML email body with reminder emphasis
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #f59e0b; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
				.content { background-color: #f9fafb; padding: 30px; border: 1px solid #e5e7eb; }
				.certificate-name { font-weight: bold; color: #1f2937; font-size: 18px; margin: 15px 0; }
				.reminder-badge { background-color: #fef3c7; color: #92400e; padding: 8px 16px; border-radius: 5px; display: inline-block; margin: 10px 0; font-weight: bold; }
				.button { display: inline-block; padding: 12px 30px; background-color: #f59e0b; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
				.button:hover { background-color: #d97706; }
				.footer { color: #6b7280; font-size: 14px; margin-top: 20px; padding-top: 20px; border-top: 1px solid #e5e7eb; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h2>🔔 Signature Reminder</h2>
				</div>
				<div class="content">
					<div class="reminder-badge">⏰ REMINDER</div>
					<p>Dear %s,</p>
					<p>This is a friendly reminder that you have a pending signature request for the following certificate:</p>
					<div class="certificate-name">"%s"</div>
					<p>Your signature is still needed. Please click the button below to review and sign the certificate:</p>
					<a href="%s" class="button">Sign Certificate Now</a>
					<p>Or copy this link to your browser:</p>
					<p style="word-break: break-all; color: #f59e0b;">%s</p>
					<div class="footer">
						<p>Best regards,<br>Easy Cert Team</p>
						<p style="font-size: 12px; color: #9ca3af;">You will continue to receive reminders until the certificate is signed. If you did not expect this email, please ignore it.</p>
					</div>
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

	for _, signerId := range signerIds {
		// Get signer details
		signer, err := signermodel.GetById(signerId)
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
		markErr := signaturemodel.MarkAsRequested(certificateId, signerId)
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
func SendAllSignaturesCompleteMail(ownerEmail, certificateName, certificateId string) error {
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", *common.Config.MailUser)
	mailer.SetHeader("To", ownerEmail)
	mailer.SetHeader("Subject", fmt.Sprintf("All Signatures Complete - %s", certificateName))

	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>
				body {
					font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
					line-height: 1.6;
					color: #333;
					margin: 0;
					padding: 0;
					background-color: #f5f5f5;
				}
				.container {
					max-width: 600px;
					margin: 40px auto;
					background-color: #ffffff;
					border-radius: 8px;
					box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
					overflow: hidden;
				}
				.header {
					background: linear-gradient(135deg, #10b981 0%%, #059669 100%%);
					color: white;
					padding: 40px 30px;
					text-align: center;
				}
				.header h1 {
					margin: 0;
					font-size: 28px;
					font-weight: 600;
				}
				.content {
					padding: 40px 30px;
				}
				.greeting {
					font-size: 18px;
					margin-bottom: 20px;
					color: #111;
				}
				.message {
					font-size: 16px;
					color: #555;
					margin-bottom: 30px;
					line-height: 1.8;
				}
				.certificate-info {
					background-color: #f0fdf4;
					border-left: 4px solid #10b981;
					padding: 20px;
					margin: 25px 0;
					border-radius: 4px;
				}
				.certificate-name {
					font-size: 18px;
					font-weight: 600;
					color: #059669;
					margin-bottom: 8px;
				}
				.certificate-id {
					font-size: 14px;
					color: #6b7280;
					font-family: 'Courier New', monospace;
				}
				.success-icon {
					font-size: 48px;
					text-align: center;
					margin: 20px 0;
				}
				.footer {
					background-color: #f9fafb;
					padding: 30px;
					text-align: center;
					font-size: 14px;
					color: #6b7280;
					border-top: 1px solid #e5e7eb;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>✅ All Signatures Complete!</h1>
				</div>
				<div class="content">
					<div class="success-icon">🎉</div>
					<p class="greeting">Great news!</p>
					<p class="message">
						All required signatures have been successfully collected for your certificate.
						The signing process is now complete and your certificate is ready.
					</p>
					<div class="certificate-info">
						<div class="certificate-name">%s</div>
						<div class="certificate-id">Certificate ID: %s</div>
					</div>
					<p class="message">
						You can now view and distribute the fully signed certificate through your dashboard.
					</p>
				</div>
				<div class="footer">
					<p>This is an automated notification from the Secure Certificate System.</p>
					<p>If you have any questions, please contact support.</p>
				</div>
			</div>
		</body>
		</html>
	`, certificateName, certificateId)

	mailer.SetBody("text/html", htmlBody)

	if err := common.Dialer.DialAndSend(mailer); err != nil {
		slog.Error("Failed to send all signatures complete email", "error", err, "recipient", ownerEmail, "certificateId", certificateId)
		return err
	}

	slog.Info("All signatures complete email sent successfully", "recipient", ownerEmail, "certificateId", certificateId)
	return nil
}
