import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import ConfirmationDialog from "./ConfirmationDialog";

describe("ConfirmationDialog", () => {
    afterEach(cleanup);

    it("focuses the safe action and closes with Escape", () => {
        const onCancel = vi.fn();
        render(
            <ConfirmationDialog
                title="Change role?"
                description="This changes administrative access."
                confirmLabel="Change role"
                onCancel={onCancel}
                onConfirm={vi.fn()}
            />,
        );

        expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();
        fireEvent.keyDown(document, { key: "Escape" });
        expect(onCancel).toHaveBeenCalledOnce();
    });

    it("locks dismissal and communicates progress while busy", () => {
        const onCancel = vi.fn();
        render(
            <ConfirmationDialog
                title="Change role?"
                description="This changes administrative access."
                confirmLabel="Change role"
                busy
                onCancel={onCancel}
                onConfirm={vi.fn()}
            />,
        );

        expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Updating…" })).toBeDisabled();
        fireEvent.keyDown(document, { key: "Escape" });
        expect(onCancel).not.toHaveBeenCalled();
    });
});
