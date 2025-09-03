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
