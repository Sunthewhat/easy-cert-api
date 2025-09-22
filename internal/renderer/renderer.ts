#!/usr/bin/env bun

// @ts-ignore - Bun provides global process
declare const process: any;

// Declare Bun global
declare const Bun: {
	write: (path: string, content: string) => Promise<void>;
};

import { fabric } from 'fabric';

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
}

interface ThumbnailRequest {
	certificate: CertificateData;
	mode: 'thumbnail';
}

interface RenderResult {
	participantId: string;
	imageBase64: string;
	status: 'success' | 'error';
	error?: string;
}

interface ThumbnailResult {
	imageBase64: string;
	status: 'success' | 'error';
	error?: string;
}

// Replace placeholders in design with participant data and clean up visual elements
function replacePlaceholders(
	designStr: string,
	participant: ParticipantData,
	qrCode?: string
): string {
	try {
		const design = JSON.parse(designStr);

		if (design.objects && Array.isArray(design.objects)) {
			console.error(`DEBUG: Processing ${design.objects.length} objects, QR code available: ${!!qrCode}`);
			design.objects = design.objects.map((obj: PlaceholderObject) => {
				// Log all objects with IDs for debugging
				if (obj.id) {
					console.error(`DEBUG: Processing object ID: ${obj.id}, type: ${obj.type}`);
				}
				// Handle grouped anchors (text anchors that are groups) - check for Group type (capital G)
				if (
					obj.isAnchor &&
					obj.id &&
					(obj.type === 'Group' || obj.type === 'group') &&
					obj.id.startsWith('PLACEHOLDER-')
				) {
					const fieldName = obj.id.replace('PLACEHOLDER-', '');
					// Check both participant.fieldName and participant.data.fieldName
					const fieldValue =
						participant[fieldName] || (participant.data && participant.data[fieldName]);

					if (fieldValue && obj.objects && Array.isArray(obj.objects)) {
						// Remove Rect objects from the group (dotted borders) and keep only textbox
						const filteredObjects = obj.objects.filter(
							(subObj: any) => subObj.type !== 'Rect' && subObj.type !== 'rect'
						);

						// Find and update the textbox within the group
						const updatedObjects = filteredObjects.map((subObj: any) => {
							if (subObj.type === 'Textbox' || subObj.type === 'textbox') {
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
					(obj.type === 'Textbox' || obj.type === 'textbox') &&
					obj.id.startsWith('PLACEHOLDER-')
				) {
					const fieldName = obj.id.replace('PLACEHOLDER-', '');
					// Check both participant.fieldName and participant.data.fieldName
					const fieldValue =
						participant[fieldName] || (participant.data && participant.data[fieldName]);

					if (fieldValue) {
						return { ...obj, text: fieldValue };
					}
				}

				// Handle simple ID-based placeholders (backward compatibility)
				else if (obj.id && obj.id.startsWith('PLACEHOLDER-')) {
					const fieldName = obj.id.replace('PLACEHOLDER-', '');
					// Check both participant.fieldName and participant.data.fieldName
					const fieldValue =
						participant[fieldName] || (participant.data && participant.data[fieldName]);
					if (fieldValue) {
						return { ...obj, text: fieldValue };
					}
				}

				// Replace QR anchor with QR code if provided - check for qr-anchor anywhere in ID
				if (obj.id && obj.id.includes('qr-anchor') && qrCode) {
					console.error(`DEBUG: Found QR anchor ${obj.id}, replacing with QR code (length: ${qrCode.length})`);
					return {
						...obj,
						type: 'Image',
						src: `data:image/png;base64,${qrCode}`,
						crossOrigin: 'anonymous',
						// Remove dotted border styling
						stroke: null,
						strokeDashArray: null,
						fill: null,
						// Make it visible and properly sized
						visible: true,
						opacity: 1,
					};
				}

				// Remove standalone dotted rect anchors that are not QR codes
				if (
					obj.id &&
					obj.id.includes('anchor') &&
					(obj.type === 'Rect' || obj.type === 'rect') &&
					obj.strokeDashArray &&
					Array.isArray(obj.strokeDashArray) &&
					!obj.id.includes('qr-anchor')
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
			(obj.type === 'Group' || obj.type === 'group') &&
			obj.objects &&
			Array.isArray(obj.objects)
		) {
			const filteredObjects = obj.objects.filter(
				(subObj: any) => subObj.type !== 'Rect' && subObj.type !== 'rect'
			);
			return { ...obj, objects: filteredObjects };
		}

		// Hide standalone dotted rect anchors
		if (
			obj.id &&
			obj.id.includes('anchor') &&
			(obj.type === 'Rect' || obj.type === 'rect') &&
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
		backgroundColor: design.background || '#ffffff',
		renderOnAddRemove: false, // Prevent automatic rendering for better performance
		enableRetinaScaling: true, // Enable retina/high-DPI scaling
	});

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
				if (object.type === 'image' && object.src) {
					// Fallback for WebP images - replace with PNG/JPG
					if (object.src.includes('.webp')) {
						object.src = object.src.replace('.webp', '.png');
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
	qrCode?: string
): Promise<RenderResult> {
	try {
		// Replace placeholders with participant data
		const processedDesign = replacePlaceholders(certificate.design, participant, qrCode);

		// Debug: Save the processed design to file
		// try {
		// const timestamp = Date.now();
		// const debugDir = '/tmp';
		// const debugFileName = `${debugDir}/processed_design_${participant.id}_${timestamp}.json`;
		// await Bun.write(debugFileName, JSON.stringify(JSON.parse(processedDesign), null, 2));
		// console.error(`DEBUG: Processed design saved to ${debugFileName}`);

		// Also save original design for comparison
		// const originalFileName = `${debugDir}/original_design_${participant.id}_${timestamp}.json`;
		// await Bun.write(
		// 	originalFileName,
		// 	JSON.stringify(JSON.parse(certificate.design), null, 2)
		// );
		// console.error(`DEBUG: Original design saved to ${originalFileName}`);

		// Save participant data
		// const participantFileName = `${debugDir}/participant_data_${participant.id}_${timestamp}.json`;
		// await Bun.write(participantFileName, JSON.stringify(participant, null, 2));
		// console.error(`DEBUG: Participant data saved to ${participantFileName}`);
		// } catch (debugError) {
		// console.error(`DEBUG: Failed to save debug files: ${debugError}`);
		// }

		// Load canvas
		const canvas = await loadCanvasWithImageFallback(processedDesign);

		// Export as base64 with high quality settings
		const imageBase64 = canvas
			.toDataURL({
				format: 'png',
				quality: 1.0, // Maximum quality
				multiplier: 2, // 2x resolution for crisp output
			})
			.split(',')[1];

		return {
			participantId: participant.id,
			imageBase64,
			status: 'success',
		};
	} catch (error) {
		return {
			participantId: participant.id,
			imageBase64: '',
			status: 'error',
			error: error instanceof Error ? error.message : 'Unknown error',
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
			backgroundColor: design.background || '#ffffff',
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
					if (object.type === 'image' && object.src) {
						// Fallback for WebP images
						if (object.src.includes('.webp')) {
							object.src = object.src.replace('.webp', '.png');
						}
					}
				}
			);
		});

		// Calculate thumbnail dimensions maintaining aspect ratio
		const maxThumbnailWidth = 200;
		const maxThumbnailHeight = 150;

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

		// Export with proper scaling and compression
		const imageBase64 = canvas
			.toDataURL({
				format: 'jpeg', // Use JPEG for smaller file size
				quality: 0.7, // Compress to 70% quality
				multiplier: scaleFactor, // Scale down to thumbnail size
			})
			.split(',')[1];

		return {
			imageBase64,
			status: 'success',
		};
	} catch (error) {
		return {
			imageBase64: '',
			status: 'error',
			error: error instanceof Error ? error.message : 'Unknown error',
		};
	}
}

// Main processing function
async function processRenderRequest(request: RenderRequest): Promise<RenderResult[]> {
	const results: RenderResult[] = [];

	for (const participant of request.participants) {
		const qrCode = request.qrCodes?.[participant.id];
		console.error(`DEBUG: Processing participant ${participant.id}, QR code available: ${!!qrCode}, QR length: ${qrCode?.length || 0}`);
		const result = await generateCertificateImage(request.certificate, participant, qrCode);
		results.push(result);
	}

	return results;
}

// Main execution
async function main() {
	try {
		// Read JSON from stdin
		const input = await new Promise<string>((resolve) => {
			let data = '';
			process.stdin.on('data', (chunk: any) => {
				data += chunk.toString();
			});
			process.stdin.on('end', () => {
				resolve(data);
			});
		});

		if (!input.trim()) {
			throw new Error('No input data received');
		}

		const request: any = JSON.parse(input);

		// Check if this is a thumbnail request
		if (request.mode === 'thumbnail') {
			const thumbnailRequest = request as ThumbnailRequest;

			if (!thumbnailRequest.certificate) {
				throw new Error('Invalid thumbnail request format: missing certificate');
			}

			// Process thumbnail rendering
			const result = await generateCertificateThumbnail(thumbnailRequest.certificate);

			// Output result as JSON to stdout
			console.log(JSON.stringify(result));
		} else {
			// Regular certificate rendering
			const renderRequest = request as RenderRequest;

			if (!renderRequest.certificate || !renderRequest.participants) {
				throw new Error('Invalid request format: missing certificate or participants');
			}

			// Process the rendering
			const results = await processRenderRequest(renderRequest);

			// Output results as JSON to stdout
			console.log(JSON.stringify(results));
		}

		process.exit(0);
	} catch (error) {
		const errorResult = {
			error: error instanceof Error ? error.message : 'Unknown error',
			status: 'error',
		};

		console.error(JSON.stringify(errorResult));
		process.exit(1);
	}
}

// Run main function
main();
