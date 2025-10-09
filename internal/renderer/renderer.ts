#!/usr/bin/env bun

// @ts-ignore - Bun provides global process
declare const process: any;

// Declare Bun global
declare const Bun: {
	write: (path: string, content: string) => Promise<void>;
};

import { fabric } from "fabric";

interface PlaceholderObject {
	id: string;
	text?: string;
	[key: string]: any;
}

interface CertificateData {
	id: string;
	design: string;
	[key: string]: any;
}

interface ParticipantData {
	id: string;
	[key: string]: any;
}

interface RenderRequest {
	certificate: CertificateData;
	participants: ParticipantData[];
	qrCodes?: { [participantId: string]: string }; // base64 QR codes from Go
	signatures?: { [signerId: string]: string }; // base64 signature images from Go
	watermark?: string; // base64 watermark image from Go
}

interface ThumbnailRequest {
	certificate: CertificateData;
	mode: "thumbnail";
}

interface RenderResult {
	participantId: string;
	imageBase64: string;
	status: "success" | "error";
	error?: string;
}

interface ThumbnailResult {
	imageBase64: string;
	status: "success" | "error";
	error?: string;
}

// Replace placeholders in design with participant data and clean up visual elements
function replacePlaceholders(
	designStr: string,
	participant: ParticipantData,
	qrCode?: string,
	signatures?: { [signerId: string]: string },
	watermark?: string
): string {
	try {
		const design = JSON.parse(designStr);

		if (design.objects && Array.isArray(design.objects)) {
			console.error(
				`DEBUG: Processing ${
					design.objects.length
				} objects, QR code available: ${!!qrCode}, signatures available: ${!!signatures}, signature count: ${
					signatures ? Object.keys(signatures).length : 0
				}`
			);

			design.objects = design.objects.map((obj: PlaceholderObject) => {
				// Log all objects with IDs for debugging
				if (obj.id) {
					console.error(`DEBUG: Processing object ID: ${obj.id}, type: ${obj.type}`);
				}
				// Handle grouped anchors (text anchors that are groups) - check for Group type (capital G)
				if (
					obj.isAnchor &&
					obj.id &&
					(obj.type === "Group" || obj.type === "group") &&
					obj.id.startsWith("PLACEHOLDER-")
				) {
					const fieldName = obj.id.replace("PLACEHOLDER-", "");
					// Check both participant.fieldName and participant.data.fieldName
					const fieldValue =
						participant[fieldName] || (participant.data && participant.data[fieldName]);

					if (fieldValue && obj.objects && Array.isArray(obj.objects)) {
						// Remove Rect objects from the group (dotted borders) and keep only textbox
						const filteredObjects = obj.objects.filter(
							(subObj: any) => subObj.type !== "Rect" && subObj.type !== "rect"
						);

						// Find and update the textbox within the group
						const updatedObjects = filteredObjects.map((subObj: any) => {
							if (subObj.type === "Textbox" || subObj.type === "textbox") {
								return { ...subObj, text: fieldValue };
							}
							return subObj;
						});

						return { ...obj, objects: updatedObjects };
					}
				}

				// Handle individual textbox anchors - check for both Textbox and textbox
				else if (
					obj.isAnchor &&
					obj.id &&
					(obj.type === "Textbox" || obj.type === "textbox") &&
					obj.id.startsWith("PLACEHOLDER-")
				) {
					const fieldName = obj.id.replace("PLACEHOLDER-", "");
					// Check both participant.fieldName and participant.data.fieldName
					const fieldValue =
						participant[fieldName] || (participant.data && participant.data[fieldName]);

					if (fieldValue) {
						return { ...obj, text: fieldValue };
					}
				}

				// Handle simple ID-based placeholders (backward compatibility)
				else if (obj.id && obj.id.startsWith("PLACEHOLDER-")) {
					const fieldName = obj.id.replace("PLACEHOLDER-", "");
					// Check both participant.fieldName and participant.data.fieldName
					const fieldValue =
						participant[fieldName] || (participant.data && participant.data[fieldName]);
					if (fieldValue) {
						return { ...obj, text: fieldValue };
					}
				}

				// Replace QR anchor with QR code if provided - check for qr-anchor anywhere in ID
				if (obj.id && obj.id.includes("qr-anchor") && qrCode) {
					console.error(
						`DEBUG: Found QR anchor ${obj.id}, replacing with QR code (length: ${qrCode.length})`
					);
					return {
						...obj,
						type: "Image",
						src: `data:image/png;base64,${qrCode}`,
						crossOrigin: "anonymous",
						// Remove dotted border styling
						stroke: null,
						strokeDashArray: null,
						fill: null,
						// Make it visible and properly sized
						visible: true,
						opacity: 1,
					};
				}

				// Handle SIGNATURE-{UUID} placeholders - Replace with real signature
				if (obj.id && obj.id.startsWith("SIGNATURE-") && signatures) {
					const signerId = obj.id.replace("SIGNATURE-", "");
					const signatureBase64 = signatures[signerId];

					if (signatureBase64) {
						console.error(
							`DEBUG: Found signature placeholder ${obj.id}, replacing with signature image (length: ${signatureBase64.length})`
						);

						// Get placeholder dimensions and position - matching test.json structure
						const placeholderLeft = obj.left || 0;
						const placeholderTop = obj.top || 0;
						const placeholderWidth = obj.width || 320;
						const placeholderHeight = obj.height || 180;
						const placeholderScaleX = obj.scaleX || 1;
						const placeholderScaleY = obj.scaleY || 1;
						const placeholderOriginX = obj.originX || "left";
						const placeholderOriginY = obj.originY || "top";

						console.error(
							`DEBUG: Placeholder - left: ${placeholderLeft}, top: ${placeholderTop}, width: ${placeholderWidth}, height: ${placeholderHeight}, scaleX: ${placeholderScaleX}, scaleY: ${placeholderScaleY}, originX: ${placeholderOriginX}, originY: ${placeholderOriginY}`
						);

						// Calculate the center point offset for objects within the group
						// Based on test.json structure where Rect is at -161, -91 (half of 320x180)
						const centerOffsetX = -(placeholderWidth / 2);
						const centerOffsetY = -(placeholderHeight / 2);

						// Signature image from frontend is always 800x450 (16:9 landscape)
						const SIGNATURE_NATIVE_WIDTH = 800;
						const SIGNATURE_NATIVE_HEIGHT = 450;

						// Calculate scale factor to fit signature into placeholder
						// Since both are 16:9, we can use either dimension
						const imageScale = placeholderWidth / SIGNATURE_NATIVE_WIDTH;

						console.error(
							`DEBUG: Signature scaling - native: ${SIGNATURE_NATIVE_WIDTH}x${SIGNATURE_NATIVE_HEIGHT}, placeholder: ${placeholderWidth}x${placeholderHeight}, scale: ${imageScale}`
						);

						// Create signature image matching the structure from test.json
						const signatureImage = {
							cropX: 0,
							cropY: 0,
							type: "Image",
							version: "6.7.1",
							originX: placeholderOriginX,
							originY: placeholderOriginY,
							left: centerOffsetX, // Offset to center in group
							top: centerOffsetY, // Offset to center in group
							width: SIGNATURE_NATIVE_WIDTH, // Use native dimensions
							height: SIGNATURE_NATIVE_HEIGHT, // Use native dimensions
							fill: "rgb(0,0,0)",
							stroke: null,
							strokeWidth: 0,
							strokeDashArray: null,
							strokeLineCap: "butt",
							strokeDashOffset: 0,
							strokeLineJoin: "miter",
							strokeUniform: false,
							strokeMiterLimit: 4,
							scaleX: imageScale, // Scale to fit placeholder
							scaleY: imageScale, // Scale to fit placeholder
							angle: 0,
							flipX: false,
							flipY: false,
							opacity: 1,
							shadow: null,
							visible: true,
							backgroundColor: "",
							fillRule: "nonzero",
							paintFirst: "fill",
							globalCompositeOperation: "source-over",
							skewX: 0,
							skewY: 0,
							src: `data:image/png;base64,${signatureBase64}`,
							crossOrigin: "anonymous",
							filters: [],
						};

						// Add watermark image overlay (if watermark is provided)
						const children: any[] = [signatureImage];

						if (watermark) {
							// Calculate watermark size - smaller scale
							const watermarkWidth = placeholderWidth * 0.45;
							const watermarkScale = watermarkWidth / 512; // Scale from native 512px width

							const watermarkImage = {
								cropX: 0,
								cropY: 0,
								type: "Image",
								version: "6.7.1",
								originX: "center",
								originY: "center",
								left: 0, // Center of group
								top: 0, // Center of group
								width: 756, // Native width
								height: 193, // Native height
								scaleX: watermarkScale,
								scaleY: watermarkScale,
								angle: 0, // No tilt - straight watermark
								opacity: 0.3,
								visible: true,
								fill: "rgb(0,0,0)",
								stroke: null,
								strokeWidth: 0,
								backgroundColor: "",
								fillRule: "nonzero",
								paintFirst: "fill",
								globalCompositeOperation: "source-over",
								skewX: 0,
								skewY: 0,
								src: `data:image/png;base64,${watermark}`,
								crossOrigin: "anonymous",
								filters: [],
							};

							children.push(watermarkImage);

							console.error(
								`DEBUG: Watermark image added - width: ${watermarkWidth}, scale: ${watermarkScale}, opacity: 0.3`
							);
						}

						console.error(
							`DEBUG: Signature image created - left: ${centerOffsetX}, top: ${centerOffsetY}, children count: ${children.length}`
						);

						// Return a complete Group matching test.json structure
						return {
							...obj, // Keep all original properties
							objects: children, // Replace children with signature + optional watermark
						};
					}
				}

				// Remove standalone dotted rect anchors that are not QR codes
				if (
					obj.id &&
					obj.id.includes("anchor") &&
					(obj.type === "Rect" || obj.type === "rect") &&
					obj.strokeDashArray &&
					Array.isArray(obj.strokeDashArray) &&
					!obj.id.includes("qr-anchor")
				) {
					// This is likely a dotted border anchor - hide it in final render
					return { ...obj, visible: false };
				}

				return obj;
			});

			// Filter out invisible objects to clean up the design
			design.objects = design.objects.filter((obj: any) => obj.visible !== false);
		}

		return JSON.stringify(design);
	} catch (error) {
		throw new Error(`Failed to replace placeholders: ${error}`);
	}
}

// Clean design for thumbnail by removing visual aids like dotted anchors
function cleanDesignForThumbnail(design: any): any {
	if (!design.objects || !Array.isArray(design.objects)) {
		return design;
	}

	const cleanedObjects = design.objects.map((obj: any) => {
		// Handle grouped anchors - remove Rect objects from groups
		if (
			(obj.type === "Group" || obj.type === "group") &&
			obj.objects &&
			Array.isArray(obj.objects)
		) {
			const filteredObjects = obj.objects.filter(
				(subObj: any) => subObj.type !== "Rect" && subObj.type !== "rect"
			);
			return { ...obj, objects: filteredObjects };
		}

		// Hide standalone dotted rect anchors
		if (
			obj.id &&
			obj.id.includes("anchor") &&
			(obj.type === "Rect" || obj.type === "rect") &&
			obj.strokeDashArray &&
			Array.isArray(obj.strokeDashArray)
		) {
			return { ...obj, visible: false };
		}

		return obj;
	});

	// Filter out invisible objects
	const visibleObjects = cleanedObjects.filter((obj: any) => obj.visible !== false);

	return { ...design, objects: visibleObjects };
}

// Load canvas with fallback handling
async function loadCanvasWithImageFallback(designStr: string): Promise<fabric.Canvas> {
	const design = JSON.parse(designStr);

	// Create fabric canvas with high-quality settings
	const canvas = new fabric.Canvas(null, {
		width: design.width || 850,
		height: design.height || 600,
		backgroundColor: design.background || "#ffffff",
		renderOnAddRemove: false, // Prevent automatic rendering for better performance
		enableRetinaScaling: true, // Enable retina/high-DPI scaling
	});

	// Add font fallbacks for production environment
	if (design.objects && Array.isArray(design.objects)) {
		design.objects = design.objects.map((obj: any) => {
			if (
				obj.type === "Textbox" ||
				obj.type === "textbox" ||
				obj.type === "Text" ||
				obj.type === "text"
			) {
				// Ensure font fallbacks for production
				if (obj.fontFamily && !obj.fontFamily.includes(",")) {
					obj.fontFamily = `${obj.fontFamily}, Arial, sans-serif`;
				} else if (!obj.fontFamily) {
					obj.fontFamily = "Arial, sans-serif";
				}
				console.error(`DEBUG: Text object font: ${obj.fontFamily}, text: "${obj.text}"`);
			}
			return obj;
		});
	}

	// Load design into canvas
	await new Promise<void>((resolve) => {
		canvas.loadFromJSON(
			design,
			() => {
				canvas.renderAll();
				resolve();
			},
			(_o: any, object: any) => {
				// Handle individual object loading errors
				if (object.type === "image" && object.src) {
					// Fallback for WebP images - replace with PNG/JPG
					if (object.src.includes(".webp")) {
						object.src = object.src.replace(".webp", ".png");
					}
				}
			}
		);
	});

	return canvas;
}

// Generate certificate image for single participant
async function generateCertificateImage(
	certificate: CertificateData,
	participant: ParticipantData,
	qrCode?: string,
	signatures?: { [signerId: string]: string },
	watermark?: string
): Promise<RenderResult> {
	try {
		// Replace placeholders with participant data
		const processedDesign = replacePlaceholders(
			certificate.design,
			participant,
			qrCode,
			signatures,
			watermark
		);

		// Load canvas
		const canvas = await loadCanvasWithImageFallback(processedDesign);

		// Export as base64 with high quality settings
		const imageBase64 = canvas
			.toDataURL({
				format: "png",
				quality: 1.0, // Maximum quality
				multiplier: 2, // 2x resolution for crisp output
			})
			.split(",")[1];

		return {
			participantId: participant.id,
			imageBase64,
			status: "success",
		};
	} catch (error) {
		return {
			participantId: participant.id,
			imageBase64: "",
			status: "error",
			error: error instanceof Error ? error.message : "Unknown error",
		};
	}
}

// Generate certificate thumbnail (no participant data, fixed small size)
async function generateCertificateThumbnail(
	certificate: CertificateData
): Promise<ThumbnailResult> {
	try {
		// Use original design without placeholder replacement
		const design = JSON.parse(certificate.design);

		// Get original dimensions or use defaults
		const originalWidth = design.width || 850;
		const originalHeight = design.height || 600;

		// Create canvas with original dimensions first and high-quality settings
		const canvas = new fabric.Canvas(null, {
			width: originalWidth,
			height: originalHeight,
			// backgroundColor: design.background || '#ffffff',
			backgroundColor: "#ffffff",
			renderOnAddRemove: false, // Prevent automatic rendering for better performance
			enableRetinaScaling: true, // Enable retina/high-DPI scaling
		});

		// Clean up design for thumbnail (remove dotted anchors, etc.)
		const cleanedDesign = cleanDesignForThumbnail(design);

		// Load cleaned design into canvas
		await new Promise<void>((resolve) => {
			canvas.loadFromJSON(
				cleanedDesign,
				() => {
					canvas.renderAll();
					resolve();
				},
				(_o: any, object: any) => {
					// Handle individual object loading errors for thumbnails
					if (object.type === "image" && object.src) {
						// Fallback for WebP images
						if (object.src.includes(".webp")) {
							object.src = object.src.replace(".webp", ".png");
						}
					}
				}
			);
		});

		// Calculate thumbnail dimensions maintaining aspect ratio
		const maxThumbnailWidth = 600;
		const maxThumbnailHeight = 600;

		const aspectRatio = originalWidth / originalHeight;
		let thumbnailWidth = maxThumbnailWidth;
		let thumbnailHeight = maxThumbnailWidth / aspectRatio;

		if (thumbnailHeight > maxThumbnailHeight) {
			thumbnailHeight = maxThumbnailHeight;
			thumbnailWidth = maxThumbnailHeight * aspectRatio;
		}

		// Calculate scale factor
		const scaleFactor = Math.min(
			thumbnailWidth / originalWidth,
			thumbnailHeight / originalHeight
		);

		// Export with proper scaling - use PNG to avoid black background issues
		const imageBase64 = canvas
			.toDataURL({
				format: "png", // Use PNG to preserve transparency and avoid black backgrounds
				quality: 1.0, // Maximum quality for PNG
				multiplier: scaleFactor, // Scale down to thumbnail size
			})
			.split(",")[1];

		return {
			imageBase64,
			status: "success",
		};
	} catch (error) {
		return {
			imageBase64: "",
			status: "error",
			error: error instanceof Error ? error.message : "Unknown error",
		};
	}
}

// Helper function to process items in batches
async function processBatch<T, R>(
	items: T[],
	batchSize: number,
	processor: (item: T) => Promise<R>
): Promise<R[]> {
	const results: R[] = [];

	for (let i = 0; i < items.length; i += batchSize) {
		const batch = items.slice(i, i + batchSize);
		console.error(
			`DEBUG: Processing batch ${Math.floor(i / batchSize) + 1}/${Math.ceil(
				items.length / batchSize
			)} (${batch.length} items)`
		);

		const batchPromises = batch.map(processor);
		const batchResults = await Promise.all(batchPromises);
		results.push(...batchResults);

		console.error(
			`DEBUG: Completed batch ${Math.floor(i / batchSize) + 1}, ${
				batchResults.length
			} certificates generated`
		);
	}

	return results;
}

// Main processing function with smart parallelization
async function processRenderRequest(request: RenderRequest): Promise<RenderResult[]> {
	const participantCount = request.participants.length;
	console.error(`DEBUG: Starting processing of ${participantCount} participants`);

	// Determine batch size based on participant count
	// For small batches: process all in parallel
	// For large batches: use smaller batches to prevent memory issues
	const batchSize =
		participantCount <= 10
			? participantCount
			: Math.min(10, Math.max(5, Math.ceil(participantCount / 4)));

	console.error(`DEBUG: Using batch size: ${batchSize} for ${participantCount} participants`);

	const processor = async (participant: ParticipantData): Promise<RenderResult> => {
		const qrCode = request.qrCodes?.[participant.id];
		console.error(
			`DEBUG: Processing participant ${
				participant.id
			}, QR code available: ${!!qrCode}, QR length: ${
				qrCode?.length || 0
			}, signatures available: ${!!request.signatures}, watermark available: ${!!request.watermark}`
		);
		return generateCertificateImage(
			request.certificate,
			participant,
			qrCode,
			request.signatures,
			request.watermark
		);
	};

	const results = await processBatch(request.participants, batchSize, processor);
	console.error(`DEBUG: Completed all processing, ${results.length} total results`);

	return results;
}

// Main execution
async function main() {
	try {
		// Read JSON from stdin
		const input = await new Promise<string>((resolve) => {
			let data = "";
			process.stdin.on("data", (chunk: any) => {
				data += chunk.toString();
			});
			process.stdin.on("end", () => {
				resolve(data);
			});
		});

		if (!input.trim()) {
			throw new Error("No input data received");
		}

		const request: any = JSON.parse(input);

		// Check if this is a thumbnail request
		if (request.mode === "thumbnail") {
			const thumbnailRequest = request as ThumbnailRequest;

			if (!thumbnailRequest.certificate) {
				throw new Error("Invalid thumbnail request format: missing certificate");
			}

			// Process thumbnail rendering
			const result = await generateCertificateThumbnail(thumbnailRequest.certificate);

			// Output result as JSON to stdout
			console.log(JSON.stringify(result));
		} else {
			// Regular certificate rendering
			const renderRequest = request as RenderRequest;

			if (!renderRequest.certificate || !renderRequest.participants) {
				throw new Error("Invalid request format: missing certificate or participants");
			}

			// Process the rendering
			const results = await processRenderRequest(renderRequest);

			// Output results as JSON to stdout
			console.log(JSON.stringify(results));
		}

		process.exit(0);
	} catch (error) {
		const errorResult = {
			error: error instanceof Error ? error.message : "Unknown error",
			status: "error",
		};

		console.error(JSON.stringify(errorResult));
		process.exit(1);
	}
}

// Run main function
main();
