import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import ReceiptEditor from "./ReceiptEditor";

const closeBitmap = vi.fn();

function canvasContext() {
    return {
        drawImage: vi.fn(),
        fillRect: vi.fn(),
        restore: vi.fn(),
        rotate: vi.fn(),
        save: vi.fn(),
        scale: vi.fn(),
        translate: vi.fn(),
        fillStyle: "",
    };
}

describe("ReceiptEditor", () => {
    beforeEach(() => {
        closeBitmap.mockClear();
        vi.stubGlobal("createImageBitmap", vi.fn().mockResolvedValue({
            width: 1200,
            height: 800,
            close: closeBitmap,
        }));
        vi.stubGlobal("URL", {
            ...URL,
            createObjectURL: vi.fn(() => "blob:prepared-receipt"),
            revokeObjectURL: vi.fn(),
        });
        vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockImplementation(() => canvasContext() as never);
        vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation((callback) => {
            callback(new Blob(["prepared jpeg"], { type: "image/jpeg" }));
        });
    });

    afterEach(() => {
        cleanup();
        vi.restoreAllMocks();
        vi.unstubAllGlobals();
    });

    it("offers distinct rear-camera and photo-library inputs", () => {
        render(<ReceiptEditor />);
        const camera = screen.getByLabelText("Take photo");
        const library = screen.getByLabelText("Choose photo");
        expect(camera).toHaveAttribute("accept", "image/*");
        expect(camera).toHaveAttribute("capture", "environment");
        expect(library).not.toHaveAttribute("capture");
        expect(screen.getByRole("button", { name: "Continue with manual entry" })).toHaveClass("min-h-11");
    });

    it("supports precise non-drag mask editing, history, and cleanup", async () => {
        const onCancel = vi.fn();
        render(<ReceiptEditor onCancel={onCancel} />);
        fireEvent.change(screen.getByLabelText("Choose photo"), {
            target: { files: [new File(["photo"], "receipt.jpg", { type: "image/jpeg" })] },
        });

        expect(await screen.findByRole("heading", { name: "Prepare your receipt" })).toBeInTheDocument();
        fireEvent.click(screen.getByRole("button", { name: /^Precise controls/ }));
        fireEvent.click(screen.getByRole("button", { name: "Add mask" }));
        expect(screen.getByText("1 permanent region")).toBeInTheDocument();
        expect(screen.getByLabelText("Selected mask left percentage")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Move mask right" })).toHaveClass("min-h-11");

        const desktopHistory = screen.getAllByRole("toolbar", { name: "Receipt editing history" })[0];
        if (!desktopHistory) throw new Error("desktop history toolbar is missing");
        fireEvent.click(within(desktopHistory).getByRole("button", { name: "Undo" }));
        expect(screen.getByText("0 permanent regions")).toBeInTheDocument();
        fireEvent.click(within(desktopHistory).getByRole("button", { name: "Redo" }));
        expect(screen.getByText("1 permanent region")).toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
        expect(onCancel).toHaveBeenCalledOnce();
        expect(closeBitmap).toHaveBeenCalledOnce();
        expect(screen.getByRole("heading", { name: "Prepare a receipt photo" })).toBeInTheDocument();
    });

    it("scans the prepared JPEG in one action and preserves edits for another scan", async () => {
        const onPrepared = vi.fn();
        render(<ReceiptEditor onPrepared={onPrepared} />);
        fireEvent.change(screen.getByLabelText("Choose photo"), {
            target: { files: [new File(["photo"], "receipt.jpg", { type: "image/jpeg" })] },
        });
        await screen.findByRole("heading", { name: "Prepare your receipt" });
        fireEvent.click(screen.getByRole("button", { name: /^Precise controls/ }));
        fireEvent.click(screen.getByRole("button", { name: "Add mask" }));
        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        await waitFor(() => expect(onPrepared).toHaveBeenCalledOnce());
        expect(screen.queryByRole("button", { name: "Use prepared receipt" })).not.toBeInTheDocument();
        const result = onPrepared.mock.calls[0][0];
        expect(result.blob.type).toBe("image/jpeg");
        expect(result.width).toBe(1200);

        const canvas = screen.getByRole("img", { name: "Receipt editing canvas" });
        await waitFor(() => expect(canvas).toHaveAttribute("width", "1200"));
        expect(canvas).toHaveAttribute("height", "800");
        expect(screen.getByText("1 permanent region")).toBeInTheDocument();
    });

    it("provides compact mobile tools and distinguishes fit from zoom", async () => {
        render(<ReceiptEditor />);
        fireEvent.change(screen.getByLabelText("Choose photo"), {
            target: { files: [new File(["photo"], "receipt.jpg", { type: "image/jpeg" })] },
        });
        await screen.findByRole("heading", { name: "Prepare your receipt" });

        const toolbar = within(screen.getByTestId("mobile-receipt-toolbar"));
        expect(toolbar.getByRole("button", { name: "Crop mode" })).toHaveClass("min-h-12");
        expect(toolbar.getByRole("button", { name: "Mask mode" })).toHaveClass("min-h-12");
        expect(toolbar.getByRole("button", { name: "Pan mode" })).toHaveClass("min-h-12");
        expect(toolbar.getByRole("button", { name: "Rotate receipt 90 degrees" })).toHaveClass("size-11");
        expect(screen.getByTestId("receipt-viewport-status")).toHaveTextContent("Fit");
        const preciseControls = screen.getByRole("button", { name: /^Precise controls/ });
        expect(preciseControls).toHaveAttribute("aria-expanded", "false");
        expect(preciseControls).toHaveClass("min-h-12");
        fireEvent.click(preciseControls);
        expect(preciseControls).toHaveAttribute("aria-expanded", "true");
        fireEvent.click(preciseControls);
        expect(preciseControls).toHaveAttribute("aria-expanded", "false");

        fireEvent.click(toolbar.getByRole("button", { name: "Zoom in preview" }));
        expect(screen.getByTestId("receipt-viewport-status")).toHaveTextContent("125% zoom");
        fireEvent.click(toolbar.getByRole("button", { name: "Fit preview" }));
        expect(screen.getByTestId("receipt-viewport-status")).toHaveTextContent("Fit");
    });

    it("cancels in-progress export without discarding edits", async () => {
        vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation(() => undefined);
        render(<ReceiptEditor />);
        fireEvent.change(screen.getByLabelText("Choose photo"), {
            target: { files: [new File(["photo"], "receipt.jpg", { type: "image/jpeg" })] },
        });
        await screen.findByRole("heading", { name: "Prepare your receipt" });
        fireEvent.click(screen.getByRole("button", { name: /^Precise controls/ }));
        fireEvent.click(screen.getByRole("button", { name: "Add mask" }));
        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        fireEvent.click(screen.getByRole("button", { name: "Cancel preparation" }));
        expect(screen.getByRole("heading", { name: "Prepare your receipt" })).toBeInTheDocument();
        expect(screen.getByText("1 permanent region")).toBeInTheDocument();
        expect(screen.getByRole("status")).toHaveTextContent("Receipt preparation cancelled");
    });

    it("keeps a failed photo retry on the receipt page", async () => {
        vi.stubGlobal("createImageBitmap", vi.fn().mockRejectedValue(new Error("unsupported")));
        Object.defineProperty(Image.prototype, "decode", {
            configurable: true,
            value: vi.fn().mockRejectedValue(new Error("unsupported")),
        });
        const onCancel = vi.fn();
        const onPhotoCleared = vi.fn();
        render(<ReceiptEditor onCancel={onCancel} onPhotoCleared={onPhotoCleared} />);
        fireEvent.change(screen.getByLabelText("Choose photo"), {
            target: { files: [new File(["photo"], "receipt.heic", { type: "image/heic" })] },
        });
        expect(await screen.findByRole("heading", { name: "We could not prepare this photo" })).toBeInTheDocument();
        fireEvent.click(screen.getByRole("button", { name: "Choose another photo" }));
        expect(screen.getByRole("heading", { name: "Prepare a receipt photo" })).toBeInTheDocument();
        expect(onCancel).not.toHaveBeenCalled();
        expect(onPhotoCleared).toHaveBeenCalledOnce();
        Reflect.deleteProperty(Image.prototype, "decode");
    });

    it("offers retry and manual entry when native decoding is unsupported", async () => {
        vi.stubGlobal("createImageBitmap", vi.fn().mockRejectedValue(new Error("unsupported")));
        Object.defineProperty(Image.prototype, "decode", {
            configurable: true,
            value: vi.fn().mockRejectedValue(new Error("unsupported")),
        });
        const onManualEntry = vi.fn();
        render(<ReceiptEditor onManualEntry={onManualEntry} />);
        fireEvent.change(screen.getByLabelText("Choose photo"), {
            target: { files: [new File(["photo"], "receipt.heic", { type: "image/heic" })] },
        });
        expect(await screen.findByRole("heading", { name: "We could not prepare this photo" })).toBeInTheDocument();
        expect(screen.getByRole("alert")).toHaveTextContent("JPEG or PNG");
        fireEvent.click(screen.getByRole("button", { name: "Manual entry" }));
        expect(onManualEntry).toHaveBeenCalledOnce();
        expect(URL.revokeObjectURL).toHaveBeenCalled();
        Reflect.deleteProperty(Image.prototype, "decode");
    });
});
