package renderer

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	digitorus_pdf "github.com/digitorus/pdf"
	"github.com/digitorus/pdfsign/sign"
	"github.com/sunthewhat/easy-cert-api/common"
)

type CertificateSigner struct {
	certificate *x509.Certificate
	privateKey  *rsa.PrivateKey
	enabled     bool
}

func NewCertificateSigner() (*CertificateSigner, error) {
	// Check if signing is enabled
	if common.Config.SigningEnabled == nil || !*common.Config.SigningEnabled {
		slog.Info("PDF signing disabled in configuration")
		return &CertificateSigner{enabled: false}, nil
	}

	// Validate required configuration
	if common.Config.SigningCertPath == nil || common.Config.SigningKeyPath == nil {
		return nil, fmt.Errorf("signing enabled but certificate or key path not configured")
	}

	certPath := *common.Config.SigningCertPath
	keyPath := *common.Config.SigningKeyPath

	// Load certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file %s: %w", certPath, err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM from %s", certPath)
	}

	certificate, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Load private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file %s: %w", keyPath, err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM from %s", keyPath)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8 format as fallback
		key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA format")
		}
	}

	slog.Info("Certificate signer initialized successfully",
		"cert_subject", certificate.Subject.String(),
		"cert_expiry", certificate.NotAfter)

	return &CertificateSigner{
		certificate: certificate,
		privateKey:  privateKey,
		enabled:     true,
	}, nil
}

func (s *CertificateSigner) SignPDF(pdfBytes []byte, certificateID, participantID string) ([]byte, error) {
	if !s.enabled {
		slog.Debug("PDF signing disabled, returning unsigned PDF", "cert_id", certificateID, "participant_id", participantID)
		return pdfBytes, nil
	}

	// Add nil checks
	if s.privateKey == nil {
		slog.Error("Private key is nil", "cert_id", certificateID, "participant_id", participantID)
		return pdfBytes, nil
	}
	if s.certificate == nil {
		slog.Error("Certificate is nil", "cert_id", certificateID, "participant_id", participantID)
		return pdfBytes, nil
	}
	if len(pdfBytes) == 0 {
		slog.Error("PDF bytes are empty", "cert_id", certificateID, "participant_id", participantID)
		return pdfBytes, fmt.Errorf("empty PDF bytes")
	}

	slog.Info("Preparing PDF signing data",
		"cert_id", certificateID,
		"participant_id", participantID,
		"private_key_size", s.privateKey.Size(),
		"cert_subject", s.certificate.Subject.String())

	// Prepare signing data
	signData := sign.SignData{
		Signature: sign.SignDataSignature{
			Info: sign.SignDataSignatureInfo{
				Name:     "Easy Cert System",
				Location: "Digital Certificate Platform",
				Reason:   fmt.Sprintf("Certificate validation for participant %s", participantID),
				Date:     time.Now(),
			},
			CertType:    sign.CertificationSignature,
			DocMDPPerm:  sign.AllowFillingExistingFormFieldsAndSignaturesPerms,
		},
		Signer:      s.privateKey,
		Certificate: s.certificate,
	}

	slog.Info("SignData prepared successfully",
		"cert_id", certificateID,
		"participant_id", participantID)

	// Create input and output buffers
	inputReader := bytes.NewReader(pdfBytes)
	var outputBuffer bytes.Buffer

	slog.Info("Starting PDF signing process",
		"cert_id", certificateID,
		"participant_id", participantID,
		"input_size", len(pdfBytes))

	// Sign the PDF using the correct API with panic recovery
	var signingError error
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic occurred during PDF signing",
					"panic", r,
					"cert_id", certificateID,
					"participant_id", participantID)
			}
		}()

		slog.Info("Creating PDF reader",
			"cert_id", certificateID,
			"participant_id", participantID)

		// Create a PDF reader
		pdfReader, err := digitorus_pdf.NewReader(inputReader, int64(len(pdfBytes)))
		if err != nil {
			slog.Error("Failed to create PDF reader",
				"error", err,
				"cert_id", certificateID,
				"participant_id", participantID)
			signingError = err
			return
		}

		// Reset the reader for signing
		inputReader.Seek(0, io.SeekStart)

		slog.Info("Calling sign.Sign function",
			"cert_id", certificateID,
			"participant_id", participantID)

		signingError = sign.Sign(inputReader, &outputBuffer, pdfReader, int64(len(pdfBytes)), signData)

		slog.Info("sign.Sign function completed",
			"cert_id", certificateID,
			"participant_id", participantID,
			"error", signingError,
			"output_size", outputBuffer.Len())

		if signingError != nil {
			slog.Error("Failed to sign PDF",
				"error", signingError,
				"cert_id", certificateID,
				"participant_id", participantID)
		}
	}()

	// Check if signing was successful
	if signingError != nil || outputBuffer.Len() == 0 {
		slog.Warn("PDF signing failed or produced empty output, returning unsigned PDF",
			"cert_id", certificateID,
			"participant_id", participantID,
			"error", signingError)
		return pdfBytes, nil
	}

	signedPDF := outputBuffer.Bytes()
	slog.Info("PDF signed successfully",
		"cert_id", certificateID,
		"participant_id", participantID,
		"original_size", len(pdfBytes),
		"signed_size", len(signedPDF))

	return signedPDF, nil
}

func (s *CertificateSigner) IsEnabled() bool {
	return s.enabled
}