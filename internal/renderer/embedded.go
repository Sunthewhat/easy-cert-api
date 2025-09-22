package renderer

import (
	"archive/zip"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/minio/minio-go/v7"
	"github.com/skip2/go-qrcode"
	"github.com/sunthewhat/easy-cert-api/common"
)

//go:embed renderer.ts
var rendererScript string

//go:embed package.json
var packageJSON string

type RenderRequest struct {
	Certificate  any               `json:"certificate"`
	Participants []any             `json:"participants"`
	QRCodes      map[string]string `json:"qrCodes,omitempty"`
}

type ThumbnailRequest struct {
	Certificate any    `json:"certificate"`
	Mode        string `json:"mode"`
}

type RenderResult struct {
	ParticipantID string `json:"participantId"`
	ImageBase64   string `json:"imageBase64"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

type ThumbnailResult struct {
	ImageBase64 string `json:"imageBase64"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

type CertificateResult struct {
	ParticipantID string `json:"participantId"`
	FilePath      string `json:"filePath"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

type EmbeddedRenderer struct {
	rendererDir string
	minIO       *minio.Client
	signer      *CertificateSigner
}

func NewEmbeddedRenderer() (*EmbeddedRenderer, error) {
	// Initialize PDF signer
	signer, err := NewCertificateSigner()
	if err != nil {
		slog.Warn("Failed to initialize PDF signer, signatures will be disabled", "error", err)
		signer = &CertificateSigner{enabled: false}
	}

	// Try Docker pre-installed path first
	dockerRendererDir := "/root/internal/renderer"
	if _, err := os.Stat(dockerRendererDir); err == nil {
		// Verify node_modules exists
		nodeModulesPath := filepath.Join(dockerRendererDir, "node_modules")
		if _, err := os.Stat(nodeModulesPath); err == nil {
			slog.Info("Using Docker pre-installed embedded renderer", "renderer_dir", dockerRendererDir)
			return &EmbeddedRenderer{
				rendererDir: dockerRendererDir,
				minIO:       common.MinIOClient,
				signer:      signer,
			}, nil
		}
	}

	// Try local development pre-installed path
	localRendererDir := "internal/renderer"
	if _, err := os.Stat(localRendererDir); err == nil {
		// Verify node_modules exists
		nodeModulesPath := filepath.Join(localRendererDir, "node_modules")
		if _, err := os.Stat(nodeModulesPath); err == nil {
			slog.Info("Using local pre-installed embedded renderer", "renderer_dir", localRendererDir)
			return &EmbeddedRenderer{
				rendererDir: localRendererDir,
				minIO:       common.MinIOClient,
				signer:      signer,
			}, nil
		}
	}

	// Final fallback - create temp directory and install fresh dependencies
	tempDir, err := os.MkdirTemp("", "easy-cert-renderer-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Write renderer script and package.json to temp directory
	if err := os.WriteFile(filepath.Join(tempDir, "renderer.ts"), []byte(rendererScript), 0644); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to write renderer script: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to write package.json: %w", err)
	}

	// Install Bun dependencies as final fallback
	slog.Info("Fallback mode - installing Bun dependencies fresh", "temp_dir", tempDir)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	installCmd := exec.CommandContext(ctx, "bun", "install")
	installCmd.Dir = tempDir
	if err := installCmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to install Bun dependencies: %w", err)
	}

	slog.Info("Fallback embedded renderer initialized", "temp_dir", tempDir)

	return &EmbeddedRenderer{
		rendererDir: tempDir,
		minIO:       common.MinIOClient,
		signer:      signer,
	}, nil
}

// Helper function to get map keys for debugging
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// QRResult represents the result of QR code generation
type QRResult struct {
	ParticipantID string
	QRCode        string
	Error         error
}

// QRJob represents a QR code generation job
type QRJob struct {
	ParticipantID string
	VerifyURL     string
	Index         int
}

func (r *EmbeddedRenderer) Close() {
	// Only cleanup if using temporary directory (fallback mode)
	if r.rendererDir != "/root/internal/renderer" && r.rendererDir != "internal/renderer" && r.rendererDir != "" {
		// This is a temporary directory, safe to remove
		os.RemoveAll(r.rendererDir)
		slog.Info("Embedded renderer cleaned up", "temp_dir", r.rendererDir)
	} else {
		slog.Info("Embedded renderer cleanup (no action needed for pre-installed directory)")
	}
}

// extractParticipantID extracts participant ID from various data types
func (r *EmbeddedRenderer) extractParticipantID(p any, index int) (string, bool) {
	slog.Info("Processing participant for QR code", "index", index, "participant_type", fmt.Sprintf("%T", p))

	var participantID string

	// Try to extract participant ID using reflection for different types
	participantValue := reflect.ValueOf(p)
	if participantValue.Kind() == reflect.Ptr {
		participantValue = participantValue.Elem()
	}

	if participantValue.Kind() == reflect.Struct {
		// Handle struct (likely CombinedParticipant)
		idField := participantValue.FieldByName("ID")
		if idField.IsValid() && idField.Kind() == reflect.String {
			participantID = idField.String()
			slog.Info("Extracted participant ID from struct", "participant_id", participantID)
		} else {
			slog.Warn("Failed to extract ID from struct", "index", index, "struct_type", participantValue.Type())
			return "", false
		}
	} else if participantMap, ok := p.(map[string]any); ok {
		// Handle map[string]any
		if id, exists := participantMap["id"].(string); exists {
			participantID = id
			slog.Info("Extracted participant ID from map", "participant_id", participantID)
		} else {
			slog.Warn("Failed to extract participant ID from map", "index", index, "participant_keys", getMapKeys(participantMap))
			return "", false
		}
	} else {
		slog.Warn("Participant is neither struct nor map", "index", index, "type", fmt.Sprintf("%T", p))
		return "", false
	}

	if participantID == "" {
		slog.Warn("Participant ID is empty", "index", index)
		return "", false
	}

	return participantID, true
}

// generateSingleQR generates a QR code for a single participant
func (r *EmbeddedRenderer) generateSingleQR(job QRJob) QRResult {
	// Generate QR code
	qrBytes, err := qrcode.Encode(job.VerifyURL, qrcode.Medium, 100)
	if err != nil {
		return QRResult{
			ParticipantID: job.ParticipantID,
			Error:         fmt.Errorf("failed to generate QR code: %w", err),
		}
	}

	// Convert to base64
	qrBase64 := base64.StdEncoding.EncodeToString(qrBytes)
	return QRResult{
		ParticipantID: job.ParticipantID,
		QRCode:        qrBase64,
	}
}

// GenerateQRCodes generates QR codes for all participants in parallel
func (r *EmbeddedRenderer) GenerateQRCodes(participants []any, certificateID string) map[string]string {
	participantCount := len(participants)
	slog.Info("Starting parallel QR code generation", "participant_count", participantCount, "certificate_id", certificateID)

	if participantCount == 0 {
		return make(map[string]string)
	}

	// Extract participant IDs and create jobs
	jobs := make([]QRJob, 0, participantCount)
	for i, p := range participants {
		participantID, ok := r.extractParticipantID(p, i)
		if !ok {
			continue
		}

		verifyURL := fmt.Sprintf("%s/validate/result/%s", *common.Config.VerifyHost, participantID)
		jobs = append(jobs, QRJob{
			ParticipantID: participantID,
			VerifyURL:     verifyURL,
			Index:         i,
		})
	}

	if len(jobs) == 0 {
		slog.Warn("No valid participants found for QR code generation")
		return make(map[string]string)
	}

	// Determine optimal number of workers (CPU cores or job count, whichever is smaller)
	numWorkers := min(runtime.NumCPU(), len(jobs))

	slog.Info("Starting QR code generation workers", "workers", numWorkers, "jobs", len(jobs))

	// Create channels
	jobChan := make(chan QRJob, len(jobs))
	resultChan := make(chan QRResult, len(jobs))

	// Start workers
	var wg sync.WaitGroup
	for i := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				slog.Info("Generating QR code", "worker", workerID, "participant_id", job.ParticipantID, "verify_url", job.VerifyURL)
				result := r.generateSingleQR(job)
				resultChan <- result
			}
		}(i)
	}

	// Send jobs to workers
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	qrCodes := make(map[string]string)
	successCount := 0
	errorCount := 0

	for result := range resultChan {
		if result.Error != nil {
			slog.Warn("Failed to generate QR code", "participant_id", result.ParticipantID, "error", result.Error)
			errorCount++
		} else {
			qrCodes[result.ParticipantID] = result.QRCode
			successCount++
		}
	}

	slog.Info("Completed parallel QR code generation",
		"certificate_id", certificateID,
		"total_jobs", len(jobs),
		"successful", successCount,
		"errors", errorCount,
		"final_qr_count", len(qrCodes))

	return qrCodes
}

func (r *EmbeddedRenderer) RenderCertificates(ctx context.Context, certificate any, participants []any) ([]RenderResult, error) {
	// Generate QR codes
	certMap, ok := certificate.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid certificate format")
	}

	certificateID, _ := certMap["id"].(string)
	qrCodes := r.GenerateQRCodes(participants, certificateID)

	// Debug: Log QR codes generation
	slog.Info("Generated QR codes", "certificate_id", certificateID, "qr_count", len(qrCodes))
	for participantID, qrCode := range qrCodes {
		slog.Info("QR code generated", "participant_id", participantID, "qr_length", len(qrCode))
	}

	// Prepare request
	request := RenderRequest{
		Certificate:  certificate,
		Participants: participants,
		QRCodes:      qrCodes,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Execute Bun renderer
	cmd := exec.CommandContext(ctx, "bun", "renderer.ts")
	cmd.Dir = r.rendererDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Bun renderer: %w", err)
	}

	// Send request data
	go func() {
		defer stdin.Close()
		stdin.Write(requestJSON)
	}()

	// Read output
	outputBytes, err := io.ReadAll(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	errorBytes, err := io.ReadAll(stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to read stderr: %w", err)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("bun renderer failed: %w, stderr: %s", err, string(errorBytes))
	}

	// Parse results
	var results []RenderResult
	if err := json.Unmarshal(outputBytes, &results); err != nil {
		return nil, fmt.Errorf("failed to parse renderer output: %w, output: %s", err, string(outputBytes))
	}

	return results, nil
}

func (r *EmbeddedRenderer) RenderThumbnail(ctx context.Context, certificate any) (*ThumbnailResult, error) {
	// Prepare thumbnail request
	request := ThumbnailRequest{
		Certificate: certificate,
		Mode:        "thumbnail",
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal thumbnail request: %w", err)
	}

	// Execute Bun renderer for thumbnail
	cmd := exec.CommandContext(ctx, "bun", "renderer.ts")
	cmd.Dir = r.rendererDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Bun renderer: %w", err)
	}

	// Send request data
	go func() {
		defer stdin.Close()
		stdin.Write(requestJSON)
	}()

	// Read output
	outputBytes, err := io.ReadAll(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	errorBytes, err := io.ReadAll(stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to read stderr: %w", err)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("bun thumbnail renderer failed: %w, stderr: %s", err, string(errorBytes))
	}

	// Parse result
	var result ThumbnailResult
	if err := json.Unmarshal(outputBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse thumbnail renderer output: %w, output: %s", err, string(outputBytes))
	}

	return &result, nil
}

func (r *EmbeddedRenderer) ProcessThumbnail(ctx context.Context, certificate any, certificateID string) (string, error) {
	bucketName := *common.Config.BucketCertificate

	// Delete all existing thumbnails for this certificate before generating new one
	r.deleteOldThumbnails(bucketName, certificateID)

	// Render thumbnail
	thumbnailResult, err := r.RenderThumbnail(ctx, certificate)
	if err != nil {
		return "", fmt.Errorf("failed to render thumbnail: %w", err)
	}

	if thumbnailResult.Status != "success" {
		return "", fmt.Errorf("thumbnail rendering failed: %s", thumbnailResult.Error)
	}

	// Decode base64 image
	imageBytes, err := base64.StdEncoding.DecodeString(thumbnailResult.ImageBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 thumbnail: %w", err)
	}

	// Generate filename with certificate ID folder (using JPEG for smaller size)
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s/thumbnail_%d_%s.jpg", certificateID, timestamp, strings.ReplaceAll(uuid.New().String(), "-", ""))

	// Ensure bucket exists and has public read policy
	if err := r.ensureBucketPublic(bucketName); err != nil {
		slog.Warn("Failed to ensure bucket is public", "error", err, "bucket", bucketName)
	}

	_, err = r.minIO.PutObject(
		context.Background(),
		bucketName,
		filename,
		bytes.NewReader(imageBytes),
		int64(len(imageBytes)),
		minio.PutObjectOptions{
			ContentType: "image/jpeg",
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to upload thumbnail to MinIO: %w", err)
	}

	// Generate the direct URL for debugging
	directURL := fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, bucketName, filename)
	slog.Info("Thumbnail uploaded to MinIO", "filename", filename, "directURL", directURL)

	return filename, nil
}

// deleteOldThumbnails removes all existing thumbnail files for a certificate
func (r *EmbeddedRenderer) deleteOldThumbnails(bucketName, certificateID string) {
	prefix := fmt.Sprintf("%s/thumbnail_", certificateID)

	objectCh := r.minIO.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	deletedCount := 0
	for object := range objectCh {
		if object.Err != nil {
			slog.Warn("Error listing thumbnail objects", "error", object.Err, "cert_id", certificateID)
			continue
		}

		err := r.minIO.RemoveObject(context.Background(), bucketName, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			slog.Warn("Failed to delete old thumbnail", "error", err, "object", object.Key, "cert_id", certificateID)
		} else {
			deletedCount++
			slog.Info("Deleted old thumbnail", "object", object.Key, "cert_id", certificateID)
		}
	}

	if deletedCount > 0 {
		slog.Info("Cleaned up old thumbnails", "count", deletedCount, "cert_id", certificateID)
	}
}

// ensureBucketPublic sets the bucket policy to allow public read access
func (r *EmbeddedRenderer) ensureBucketPublic(bucketName string) error {
	// Check if bucket exists
	exists, err := r.minIO.BucketExists(context.Background(), bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	// Create bucket if it doesn't exist
	if !exists {
		err = r.minIO.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	// Set public read policy
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::%s/*"
			}
		]
	}`, bucketName)

	err = r.minIO.SetBucketPolicy(context.Background(), bucketName, policy)
	if err != nil {
		return fmt.Errorf("failed to set bucket policy: %w", err)
	}

	return nil
}

// GenerateAccessibleURL creates a static URL for accessing MinIO objects
func (r *EmbeddedRenderer) GenerateAccessibleURL(bucketName, objectName string) string {
	// Return static URL - objects should be publicly accessible
	return fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, bucketName, objectName)
}

func (r *EmbeddedRenderer) ConvertToPDF(imageBase64 string, participantID string, certificateID string) ([]byte, error) {
	// Decode base64 image
	imageBytes, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Create temporary image file
	tempFile, err := os.CreateTemp("", "cert-*.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp image file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.Write(imageBytes); err != nil {
		return nil, fmt.Errorf("failed to write temp image: %w", err)
	}
	tempFile.Close()

	// Create PDF
	pdf := gofpdf.New("L", "mm", "A4", "") // Landscape orientation for certificates
	pdf.AddPage()

	// Get page dimensions
	pageWidth, pageHeight := pdf.GetPageSize()

	// Add image to PDF (fit to page)
	pdf.Image(tempFile.Name(), 0, 0, pageWidth, pageHeight, false, "", 0, "")

	// Output PDF to buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	pdfBytes := buf.Bytes()

	// Sign the PDF if signer is available and enabled
	if r.signer != nil && r.signer.IsEnabled() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Panic occurred during PDF signing in ConvertToPDF",
						"panic", r,
						"cert_id", certificateID,
						"participant_id", participantID)
				}
			}()

			signedPDF, err := r.signer.SignPDF(pdfBytes, certificateID, participantID)
			if err != nil {
				slog.Warn("Failed to sign PDF, returning unsigned version",
					"error", err,
					"cert_id", certificateID,
					"participant_id", participantID)
				return
			}

			if len(signedPDF) > 0 {
				pdfBytes = signedPDF
			}
		}()
	}

	return pdfBytes, nil
}

func (r *EmbeddedRenderer) UploadToMinIO(data []byte, filename string) (string, error) {
	return r.UploadToMinIOWithContentType(data, filename, "application/pdf")
}

func (r *EmbeddedRenderer) UploadToMinIOWithContentType(data []byte, filename string, contentType string) (string, error) {
	bucketName := *common.Config.BucketCertificate

	// Ensure bucket exists and has public read policy
	if err := r.ensureBucketPublic(bucketName); err != nil {
		slog.Warn("Failed to ensure bucket is public", "error", err, "bucket", bucketName)
	}

	_, err := r.minIO.PutObject(
		context.Background(),
		bucketName,
		filename,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	// Generate the direct URL for debugging
	directURL := fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, bucketName, filename)
	slog.Info("File uploaded to MinIO", "filename", filename, "contentType", contentType, "directURL", directURL)

	return filename, nil
}

func (r *EmbeddedRenderer) CreateZipArchive(results []CertificateResult) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	defer zipWriter.Close()

	for _, result := range results {
		if result.Status != "success" || result.FilePath == "" {
			continue
		}

		// Download file from MinIO
		object, err := r.minIO.GetObject(
			context.Background(),
			*common.Config.BucketCertificate,
			result.FilePath,
			minio.GetObjectOptions{},
		)
		if err != nil {
			slog.Warn("Failed to download file for ZIP", "file_path", result.FilePath, "error", err)
			continue
		}

		data, err := io.ReadAll(object)
		object.Close()
		if err != nil {
			slog.Warn("Failed to read file data for ZIP", "file_path", result.FilePath, "error", err)
			continue
		}

		// Add to ZIP
		filename := fmt.Sprintf("certificate_%s.pdf", result.ParticipantID)
		zipFile, err := zipWriter.Create(filename)
		if err != nil {
			slog.Warn("Failed to create ZIP entry", "filename", filename, "error", err)
			continue
		}

		if _, err := zipFile.Write(data); err != nil {
			slog.Warn("Failed to write ZIP entry", "filename", filename, "error", err)
			continue
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	return buf.Bytes(), nil
}

func (r *EmbeddedRenderer) ProcessCertificates(ctx context.Context, certificate any, participants []any) ([]CertificateResult, string, error) {
	// Extract certificate ID
	certMap, ok := certificate.(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("invalid certificate format for folder creation")
	}
	certificateID, _ := certMap["id"].(string)

	// Render certificates
	renderResults, err := r.RenderCertificates(ctx, certificate, participants)
	if err != nil {
		return nil, "", fmt.Errorf("failed to render certificates: %w", err)
	}

	var certificateResults []CertificateResult

	// Process each rendered certificate
	for _, renderResult := range renderResults {
		if renderResult.Status != "success" {
			certificateResults = append(certificateResults, CertificateResult{
				ParticipantID: renderResult.ParticipantID,
				Status:        "error",
				Error:         renderResult.Error,
			})
			continue
		}

		// Convert to PDF
		pdfBytes, err := r.ConvertToPDF(renderResult.ImageBase64, renderResult.ParticipantID, certificateID)
		if err != nil {
			slog.Error("Failed to convert to PDF", "participant_id", renderResult.ParticipantID, "error", err)
			certificateResults = append(certificateResults, CertificateResult{
				ParticipantID: renderResult.ParticipantID,
				Status:        "error",
				Error:         fmt.Sprintf("PDF conversion failed: %v", err),
			})
			continue
		}

		// Generate filename with certificate ID folder
		timestamp := time.Now().Unix()
		filename := fmt.Sprintf("%s/certificate_%d_%s.pdf", certificateID, timestamp, strings.ReplaceAll(uuid.New().String(), "-", ""))

		// Upload to MinIO
		filePath, err := r.UploadToMinIO(pdfBytes, filename)
		if err != nil {
			slog.Error("Failed to upload PDF", "participant_id", renderResult.ParticipantID, "error", err)
			certificateResults = append(certificateResults, CertificateResult{
				ParticipantID: renderResult.ParticipantID,
				Status:        "error",
				Error:         fmt.Sprintf("Upload failed: %v", err),
			})
			continue
		}

		certificateResults = append(certificateResults, CertificateResult{
			ParticipantID: renderResult.ParticipantID,
			FilePath:      filePath,
			Status:        "success",
		})
	}

	// Create ZIP archive
	zipBytes, err := r.CreateZipArchive(certificateResults)
	if err != nil {
		return certificateResults, "", fmt.Errorf("failed to create ZIP archive: %w", err)
	}

	// Upload ZIP to MinIO with correct content type
	timestamp := time.Now().Unix()
	zipFilename := fmt.Sprintf("%s/certificates_%d_%s.zip", certificateID, timestamp, strings.ReplaceAll(uuid.New().String(), "-", ""))

	zipFilePath, err := r.UploadToMinIOWithContentType(zipBytes, zipFilename, "application/zip")
	if err != nil {
		return certificateResults, "", fmt.Errorf("failed to upload ZIP: %w", err)
	}

	return certificateResults, zipFilePath, nil
}
