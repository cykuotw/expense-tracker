export const RECEIPT_MAX_SOURCE_BYTES = 20 * 1024 * 1024;
export const RECEIPT_MAX_OUTPUT_BYTES = 3_670_016;
export const RECEIPT_MAX_OUTPUT_DIMENSION = 4_096;
export const RECEIPT_MAX_OUTPUT_PIXELS = 16_000_000;
export const RECEIPT_MAX_DECODE_PIXELS = 80_000_000;

export type ReceiptRotation = 0 | 90 | 180 | 270;

export interface NormalizedRect {
    x: number;
    y: number;
    width: number;
    height: number;
}

export interface ReceiptMask extends NormalizedRect {
    id: string;
}

export interface ReceiptEditState {
    rotation: ReceiptRotation;
    crop: NormalizedRect;
    masks: ReceiptMask[];
}

export interface ReceiptHistory {
    past: ReceiptEditState[];
    present: ReceiptEditState;
    future: ReceiptEditState[];
}

export interface DecodedReceipt {
    source: CanvasImageSource;
    width: number;
    height: number;
    dispose: () => void;
}

export interface PreparedReceipt {
    blob: Blob;
    width: number;
    height: number;
    quality: number;
}

const MIN_RECT_SIZE = 0.01;

export const fullReceiptEditState = (): ReceiptEditState => ({
    rotation: 0,
    crop: { x: 0, y: 0, width: 1, height: 1 },
    masks: [],
});

export const createReceiptHistory = (): ReceiptHistory => ({
    past: [],
    present: fullReceiptEditState(),
    future: [],
});

const clamp = (value: number, minimum: number, maximum: number) =>
    Math.min(maximum, Math.max(minimum, value));

export function normalizeRect(rect: NormalizedRect): NormalizedRect {
    const x1 = clamp(Math.min(rect.x, rect.x + rect.width), 0, 1);
    const y1 = clamp(Math.min(rect.y, rect.y + rect.height), 0, 1);
    const x2 = clamp(Math.max(rect.x, rect.x + rect.width), 0, 1);
    const y2 = clamp(Math.max(rect.y, rect.y + rect.height), 0, 1);

    return updateRect({
        x: x1,
        y: y1,
        width: Math.max(MIN_RECT_SIZE, x2 - x1),
        height: Math.max(MIN_RECT_SIZE, y2 - y1),
    }, {});
}

export function updateRect(
    rect: NormalizedRect,
    changes: Partial<NormalizedRect>,
): NormalizedRect {
    const width = clamp(changes.width ?? rect.width, MIN_RECT_SIZE, 1);
    const height = clamp(changes.height ?? rect.height, MIN_RECT_SIZE, 1);
    const x = clamp(changes.x ?? rect.x, 0, 1 - width);
    const y = clamp(changes.y ?? rect.y, 0, 1 - height);
    return { x, y, width, height };
}

export function rectFromPoints(
    start: { x: number; y: number },
    end: { x: number; y: number },
): NormalizedRect {
    return normalizeRect({
        x: start.x,
        y: start.y,
        width: end.x - start.x,
        height: end.y - start.y,
    });
}

export function rotateRectClockwise(rect: NormalizedRect): NormalizedRect {
    return {
        x: 1 - rect.y - rect.height,
        y: rect.x,
        width: rect.height,
        height: rect.width,
    };
}

export function rotateEditStateClockwise(state: ReceiptEditState): ReceiptEditState {
    return {
        rotation: ((state.rotation + 90) % 360) as ReceiptRotation,
        crop: rotateRectClockwise(state.crop),
        masks: state.masks.map((mask) => ({
            ...rotateRectClockwise(mask),
            id: mask.id,
        })),
    };
}

export function commitReceiptEdit(
    history: ReceiptHistory,
    next: ReceiptEditState,
): ReceiptHistory {
    if (JSON.stringify(history.present) === JSON.stringify(next)) return history;
    return {
        past: [...history.past, history.present],
        present: next,
        future: [],
    };
}

export function undoReceiptEdit(history: ReceiptHistory): ReceiptHistory {
    const previous = history.past.at(-1);
    if (!previous) return history;
    return {
        past: history.past.slice(0, -1),
        present: previous,
        future: [history.present, ...history.future],
    };
}

export function redoReceiptEdit(history: ReceiptHistory): ReceiptHistory {
    const next = history.future[0];
    if (!next) return history;
    return {
        past: [...history.past, history.present],
        present: next,
        future: history.future.slice(1),
    };
}

export function orientedDimensions(
    width: number,
    height: number,
    rotation: ReceiptRotation,
) {
    return rotation === 90 || rotation === 270
        ? { width: height, height: width }
        : { width, height };
}

function throwIfAborted(signal?: AbortSignal) {
    if (signal?.aborted) throw new DOMException("Receipt processing was cancelled", "AbortError");
}

export async function decodeReceiptFile(
    file: File,
    signal?: AbortSignal,
): Promise<DecodedReceipt> {
    throwIfAborted(signal);
    if (file.size === 0) throw new Error("This image is empty. Choose another photo.");
    if (file.size > RECEIPT_MAX_SOURCE_BYTES) {
        throw new Error("This photo is larger than 20 MiB. Choose a smaller photo.");
    }

    if (typeof createImageBitmap === "function") {
        let bitmap: ImageBitmap | undefined;
        try {
            const createdBitmap = await createImageBitmap(file, { imageOrientation: "from-image" });
            bitmap = createdBitmap;
            throwIfAborted(signal);
            validateDecodedDimensions(createdBitmap.width, createdBitmap.height);
            return {
                source: createdBitmap,
                width: createdBitmap.width,
                height: createdBitmap.height,
                dispose: () => createdBitmap.close(),
            };
        } catch (error) {
            bitmap?.close();
            if (error instanceof DOMException && error.name === "AbortError") throw error;
        }
    }

    const url = URL.createObjectURL(file);
    const image = new Image();
    image.decoding = "async";
    image.src = url;
    try {
        await image.decode();
        throwIfAborted(signal);
        validateDecodedDimensions(image.naturalWidth, image.naturalHeight);
        return {
            source: image,
            width: image.naturalWidth,
            height: image.naturalHeight,
            dispose: () => {
                image.src = "";
                URL.revokeObjectURL(url);
            },
        };
    } catch (error) {
        image.src = "";
        URL.revokeObjectURL(url);
        if (error instanceof DOMException && error.name === "AbortError") throw error;
        throw new Error(
            "This browser could not open the photo. Choose a JPEG or PNG, or continue with manual entry.",
        );
    }
}

function validateDecodedDimensions(width: number, height: number) {
    if (width <= 0 || height <= 0 || width * height > RECEIPT_MAX_DECODE_PIXELS) {
        throw new Error("This photo has unsupported dimensions. Choose a smaller photo.");
    }
}

function createCanvas(width: number, height: number) {
    const canvas = document.createElement("canvas");
    canvas.width = Math.max(1, Math.round(width));
    canvas.height = Math.max(1, Math.round(height));
    return canvas;
}

function canvasBlob(canvas: HTMLCanvasElement, quality: number): Promise<Blob> {
    return new Promise((resolve, reject) => {
        canvas.toBlob((blob) => {
            if (blob) resolve(blob);
            else reject(new Error("The browser could not create the prepared receipt."));
        }, "image/jpeg", quality);
    });
}

function outputDimensions(width: number, height: number) {
    const scale = Math.min(
        1,
        RECEIPT_MAX_OUTPUT_DIMENSION / Math.max(width, height),
        Math.sqrt(RECEIPT_MAX_OUTPUT_PIXELS / (width * height)),
    );
    return {
        width: Math.max(1, Math.round(width * scale)),
        height: Math.max(1, Math.round(height * scale)),
    };
}

function drawEditedReceipt(
    canvas: HTMLCanvasElement,
    decoded: DecodedReceipt,
    edit: ReceiptEditState,
) {
    const context = canvas.getContext("2d", { alpha: false });
    if (!context) throw new Error("Canvas image editing is unavailable in this browser.");
    const oriented = orientedDimensions(decoded.width, decoded.height, edit.rotation);
    const crop = updateRect(edit.crop, edit.crop);
    const cropPixels = {
        x: crop.x * oriented.width,
        y: crop.y * oriented.height,
        width: crop.width * oriented.width,
        height: crop.height * oriented.height,
    };

    context.save();
    context.fillStyle = "#ffffff";
    context.fillRect(0, 0, canvas.width, canvas.height);
    context.scale(canvas.width / cropPixels.width, canvas.height / cropPixels.height);
    context.translate(-cropPixels.x, -cropPixels.y);

    switch (edit.rotation) {
        case 90:
            context.translate(decoded.height, 0);
            context.rotate(Math.PI / 2);
            break;
        case 180:
            context.translate(decoded.width, decoded.height);
            context.rotate(Math.PI);
            break;
        case 270:
            context.translate(0, decoded.width);
            context.rotate(-Math.PI / 2);
            break;
    }
    context.drawImage(decoded.source, 0, 0, decoded.width, decoded.height);
    context.restore();

    context.save();
    context.scale(canvas.width / cropPixels.width, canvas.height / cropPixels.height);
    context.translate(-cropPixels.x, -cropPixels.y);
    context.fillStyle = "#111111";
    for (const mask of edit.masks) {
        context.fillRect(
            mask.x * oriented.width,
            mask.y * oriented.height,
            mask.width * oriented.width,
            mask.height * oriented.height,
        );
    }
    context.restore();
}

export function renderReceiptPreview(
    canvas: HTMLCanvasElement,
    decoded: DecodedReceipt,
    rotation: ReceiptRotation,
) {
    const oriented = orientedDimensions(decoded.width, decoded.height, rotation);
    const scale = Math.min(1, 1_200 / Math.max(oriented.width, oriented.height));
    canvas.width = Math.max(1, Math.round(oriented.width * scale));
    canvas.height = Math.max(1, Math.round(oriented.height * scale));
    const context = canvas.getContext("2d", { alpha: false });
    if (!context) throw new Error("Canvas image editing is unavailable in this browser.");
    context.fillStyle = "#ffffff";
    context.fillRect(0, 0, canvas.width, canvas.height);
    context.scale(canvas.width / oriented.width, canvas.height / oriented.height);
    switch (rotation) {
        case 90:
            context.translate(decoded.height, 0);
            context.rotate(Math.PI / 2);
            break;
        case 180:
            context.translate(decoded.width, decoded.height);
            context.rotate(Math.PI);
            break;
        case 270:
            context.translate(0, decoded.width);
            context.rotate(-Math.PI / 2);
            break;
    }
    context.drawImage(decoded.source, 0, 0, decoded.width, decoded.height);
}

export async function prepareReceipt(
    decoded: DecodedReceipt,
    edit: ReceiptEditState,
    signal?: AbortSignal,
): Promise<PreparedReceipt> {
    throwIfAborted(signal);
    const oriented = orientedDimensions(decoded.width, decoded.height, edit.rotation);
    const crop = updateRect(edit.crop, edit.crop);
    let dimensions = outputDimensions(
        oriented.width * crop.width,
        oriented.height * crop.height,
    );
    let canvas = createCanvas(dimensions.width, dimensions.height);
    drawEditedReceipt(canvas, decoded, edit);

    const qualities = [0.92, 0.84, 0.76, 0.68, 0.6, 0.52];
    for (let resizeAttempt = 0; resizeAttempt < 6; resizeAttempt += 1) {
        for (const quality of qualities) {
            throwIfAborted(signal);
            const blob = await canvasBlob(canvas, quality);
            throwIfAborted(signal);
            if (blob.size <= RECEIPT_MAX_OUTPUT_BYTES) {
                const width = canvas.width;
                const height = canvas.height;
                canvas.width = 0;
                canvas.height = 0;
                return { blob, width, height, quality };
            }
        }

        const largestDimension = Math.max(canvas.width, canvas.height);
        if (largestDimension <= 640) break;
        const scale = Math.max(640 / largestDimension, 0.8);
        dimensions = {
            width: Math.max(1, Math.round(canvas.width * scale)),
            height: Math.max(1, Math.round(canvas.height * scale)),
        };
        if (dimensions.width === canvas.width && dimensions.height === canvas.height) break;
        canvas.width = 0;
        canvas.height = 0;
        canvas = createCanvas(dimensions.width, dimensions.height);
        drawEditedReceipt(canvas, decoded, edit);
    }

    canvas.width = 0;
    canvas.height = 0;
    throw new Error("The prepared receipt is still too large. Crop it more tightly and try again.");
}

export function formatReceiptBytes(bytes: number) {
    if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KiB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}
