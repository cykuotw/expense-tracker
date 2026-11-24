import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { useEffect } from "react";

let finishPreparation: (() => void) | undefined;
const { useCreateExpenseMock } = vi.hoisted(() => ({ useCreateExpenseMock: vi.fn() }));
vi.mock("@/hooks/CreateExpenseContextHooks", () => ({
    useCreateExpense: () => useCreateExpenseMock(),
}));
vi.mock("./ReceiptEditor", () => ({
    default: function MockEditor({ initialFile, onPhotoLoaded, onPhotoCleared, onPrepared }: {
        initialFile?: File | null;
        onPhotoCleared?: () => void;
        onPhotoLoaded?: () => void;
        onPrepared?: (receipt: { blob: Blob; width: number; height: number; quality: number }) => void;
    }) {
        useEffect(() => {
            if (initialFile) onPhotoLoaded?.();
        }, [initialFile, onPhotoLoaded]);
        return <div>
            <button onClick={onPhotoCleared}>Choose another photo</button>
            <button onClick={onPhotoLoaded}>Take photo</button>
            <p>Editor: {initialFile?.name ?? "no photo"}</p>
            <button onClick={() => { finishPreparation = () => onPrepared?.({ blob: new Blob(["test"]), width: 10, height: 10, quality: 1 }); }}>Start delayed scan</button>
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
        vi.stubGlobal("URL", class extends URL {
            static createObjectURL = vi.fn(() => "blob:prepared-receipt");
            static revokeObjectURL = vi.fn();
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

    it("submits the original note when preparation overlaps an edit and then rescans", async () => {
        render(<MemoryRouter><ReceiptScanPage /></MemoryRouter>);
        fireEvent.click(await screen.findByRole("button", { name: "Take photo" }));
        const note = screen.getByLabelText("Scan note (optional)");
        fireEvent.change(note, { target: { value: "First" } });
        fireEvent.click(screen.getByRole("button", { name: "Start delayed scan" }));
        fireEvent.change(note, { target: { value: "Second" } });
        await act(async () => finishPreparation?.());
        expect(startReceiptOCR).toHaveBeenLastCalledWith(expect.any(Object), "First");
        fireEvent.click(screen.getByRole("button", { name: "Edit photo" }));
        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        expect(startReceiptOCR).toHaveBeenLastCalledWith(expect.any(Object), "Second");
    });

    it("uses the existing photo input from the desktop header and keeps back navigation", async () => {
        render(<MemoryRouter initialEntries={["/create_expense/receipt?g=group-1"]}>
            <Routes>
                <Route path="/create_expense/receipt" element={<ReceiptScanPage />} />
                <Route path="/create_expense" element={<div>Manual form</div>} />
            </Routes>
        </MemoryRouter>);

        const input = screen.getByLabelText("Receipt photo file");
        const openPicker = vi.spyOn(input, "click");
        const desktopHeader = screen.getByRole("heading", { name: "Receipt photo" }).closest("header");
        if (!desktopHeader) throw new Error("desktop receipt header is missing");
        fireEvent.click(within(desktopHeader).getByRole("button", { name: "Upload photo" }));
        expect(openPicker).toHaveBeenCalledOnce();

        fireEvent.click(screen.getByRole("button", { name: "Back to expense" }));
        expect(clearReceiptWorkflow).toHaveBeenCalledOnce();
        expect(screen.getByText("Manual form")).toBeInTheDocument();
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
        fireEvent.change(screen.getByLabelText("Receipt photo file"), {
            target: { files: [new File(["first"], "first.jpg", { type: "image/jpeg" })] },
        });
        expect(await screen.findByText("Editor: first.jpg")).toBeInTheDocument();
        expect(screen.getAllByRole("button", { name: "Change photo" })[0]).toBeInTheDocument();

        fireEvent.change(screen.getByLabelText("Scan note (optional)"), { target: { value: "Fully covered by CT Money" } });
        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        expect(startReceiptOCR).toHaveBeenCalledOnce();
        expect(startReceiptOCR).toHaveBeenCalledWith(expect.any(Object), "Fully covered by CT Money");
        expect(screen.getByText("Inline review: blob:prepared-receipt")).toBeInTheDocument();
        expect(screen.getByTestId("location")).toHaveTextContent("/create_expense/receipt");

        fireEvent.click(screen.getByRole("button", { name: "Edit photo" }));
        expect(clearReceiptWorkflow).toHaveBeenCalledOnce();
        expect(screen.getByText("Editor: first.jpg")).toBeInTheDocument();

        expect(screen.getByLabelText("Scan note (optional)")).toHaveValue("Fully covered by CT Money");
        fireEvent.click(screen.getByRole("button", { name: "Choose another photo" }));
        fireEvent.click(screen.getByRole("button", { name: "Take photo" }));
        expect(screen.getByLabelText("Scan note (optional)")).toHaveValue("");

        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        fireEvent.change(screen.getByLabelText("Receipt photo file"), {
            target: { files: [new File(["second"], "second.jpg", { type: "image/jpeg" })] },
        });
        expect(await screen.findByText("Editor: second.jpg")).toBeInTheDocument();
        expect(clearReceiptWorkflow).toHaveBeenCalledTimes(2);
        await waitFor(() => expect(URL.revokeObjectURL).toHaveBeenCalled());
    });

    it("scrolls to the top after opening each scan view without resetting on unrelated renders", async () => {
        const scrollTo = vi.fn(() => {
            expect(screen.getByText("Inline review: blob:prepared-receipt")).toBeInTheDocument();
            expect(screen.getByRole("heading", { name: "Receipt photo", hidden: true }).closest("main")?.parentElement).toHaveAttribute("hidden");
        });
        vi.stubGlobal("scrollTo", scrollTo);
        const page = <MemoryRouter initialEntries={["/create_expense/receipt?g=group-1"]}>
            <ReceiptScanPage />
        </MemoryRouter>;
        const { rerender } = render(page);

        expect(await screen.findByText("Editor: no photo")).toBeInTheDocument();
        expect(scrollTo).not.toHaveBeenCalled();

        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        expect(scrollTo).toHaveBeenCalledExactlyOnceWith({ top: 0, left: 0, behavior: "instant" });

        rerender(<MemoryRouter initialEntries={["/create_expense/receipt?g=group-1"]}>
            <ReceiptScanPage />
        </MemoryRouter>);
        expect(scrollTo).toHaveBeenCalledTimes(1);

        fireEvent.click(screen.getByRole("button", { name: "Edit photo" }));
        expect(scrollTo).toHaveBeenCalledTimes(1);
        fireEvent.click(screen.getByRole("button", { name: "Scan receipt" }));
        expect(scrollTo).toHaveBeenCalledTimes(2);
    });
});
