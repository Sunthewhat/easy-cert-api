package certificate_controller

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *CertificateController) GetAnchorList(c *fiber.Ctx) error {
	certId := c.Params("certId")

	if certId == "" {
		slog.Warn("Certificate getAnchorList attempt with empty ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	cert, err := ctrl.certRepo.GetById(certId)

	if err != nil {
		slog.Error("Error getting certificate", "certId", certId, "error", err)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Getting non-existing certificate", "certId", certId)
		return response.SendFailed(c, "Certificate not found")
	}

	// Parse the certificate design JSON
	var design map[string]any
	if err := json.Unmarshal([]byte(cert.Design), &design); err != nil {
		slog.Error("Error parsing certificate design", "certId", certId, "error", err)
		return response.SendInternalError(c, err)
	}

	// Extract objects array
	objects, ok := design["objects"].([]any)
	if !ok {
		slog.Warn("Invalid design format - objects array not found", "certId", certId)
		return response.SendFailed(c, "Invalid certificate design format")
	}

	// Find all placeholder objects and extract anchor names
	var anchorNames []string
	for _, obj := range objects {
		objMap, ok := obj.(map[string]any)
		if !ok {
			continue
		}

		id, exists := objMap["id"].(string)
		if exists && strings.HasPrefix(id, "PLACEHOLDER-") {
			// Extract the anchor name after "PLACEHOLDER-"
			anchorName := strings.TrimPrefix(id, "PLACEHOLDER-")
			anchorNames = append(anchorNames, anchorName)
		}
	}

	return response.SendSuccess(c, "Anchor list retrieved successfully", anchorNames)
}
