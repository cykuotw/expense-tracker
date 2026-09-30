import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { useEffect } from "react";

const { useCreateExpenseMock } = vi.hoisted(() => ({ useCreateExpenseMock: vi.fn() }));
vi.mock("@/hooks/CreateExpenseContextHooks", () => ({
    useCreateExpense: () => useCreateExpenseMock(),
}));
vi.mock("./ReceiptEditor", () => ({
    default: function MockEditor({ initialFile, onPhotoLoaded, onPrepared }: {
        initialFile?: File | null;
        onPhotoLoaded?: () => void;
        onPrepared?: (receipt: { blob: Blob; width: number; height: number; quality: number }) => void;
    }) {
        useEffect(() => {
            if (initialFile) onPhotoLoaded?.();
        }, [initialFile, onPhotoLoaded]);
        return <div>
            <p>Editor: {initialFile?.name ?? "no photo"}</p>
            <button onClick={() => onPrepared?.({ blob: new Blob(["prepared"], { type: "image/jpeg" }), width: 800, height: 1200, quality: 0.8 })}>Scan receipt</button>
        </div>;
    },
}));
vi.mock("./ReceiptDraftReview", () => ({
    default: ({ photoURL, onEditPhoto }: { photoURL?: string | null; onEditPhoto?: () => void }) =>
        <div><p>Inline review: {photoURL}</p><button onClick={onEditPhoto}>Edit photo</button></div>,
}));

import ReceiptScanPage from "./ReceiptScanPage";

function Location() {
    return <p data-testid="location">{useLocation().pathname}</p>;
}

describe("ReceiptScanPage", () => {
    const startReceiptOCR = vi.fn();
    const clearReceiptWorkflow = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
        vi.stubGlobal("URL", {
            ...URL,
            createObjectURL: vi.fn(() => "blob:prepared-receipt"),
            revokeObjectURL: vi.fn(),
        });
        useCreateExpenseMock.mockReturnValue({
            selectedGroupId: "group-1",
            receiptOCREnabled: true,
            startReceiptOCR,
            clearReceiptWorkflow,
        });
    });

    afterEach(() => {
        cleanup();
        vi.unstubAllGlobals();
    });

    it("scans and reviews on one route, then replaces the photo without scrolling back", async () => {
        render(<MemoryRouter initialEntries={["/create_expense/receipt?g=group-1"]}>
            <Location />
            <Routes>
                <Route path="/create_expense/receipt" element={<ReceiptScanPage />} />
                <Route path="/create_expense" element={<div>Manual form</div>} />
            </Routes>
        </MemoryRouter>);

        expect(await screen.findByText("Editor: no photo")).toBeInTheDocument();
        fireEvent.change(screen.getByLabelText("Upload photo"), {
            target: { files: [new File(["first"], "first.jpg", { type: "image/jpeg" })] },
        });
        expect(await screen.findByText("Editor: first.jpg")).toBeInTheDocument();
        expect(screen.getByLabelText("Change photo")).toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        expect(startReceiptOCR).toHaveBeenCalledOnce();
        expect(screen.getByText("Inline review: blob:prepared-receipt")).toBeInTheDocument();
        expect(screen.getByTestId("location")).toHaveTextContent("/create_expense/receipt");

        fireEvent.click(screen.getByRole("button", { name: "Edit photo" }));
        expect(clearReceiptWorkflow).toHaveBeenCalledOnce();
        expect(screen.getByText("Editor: first.jpg")).toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        fireEvent.change(screen.getByLabelText("Change photo"), {
            target: { files: [new File(["second"], "second.jpg", { type: "image/jpeg" })] },
        });
        expect(await screen.findByText("Editor: second.jpg")).toBeInTheDocument();
        expect(clearReceiptWorkflow).toHaveBeenCalledTimes(2);
        await waitFor(() => expect(URL.revokeObjectURL).toHaveBeenCalled());
    });
});
