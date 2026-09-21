import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import AppErrorBoundary from "./AppErrorBoundary";

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

function BrokenView(): never {
    throw new Error("private-expense-token");
}

describe("AppErrorBoundary", () => {
    it("shows an accessible recovery screen without exception details", () => {
        vi.spyOn(console, "error").mockImplementation(() => undefined);
        const onError = vi.fn();

        render(
            <AppErrorBoundary onError={onError}>
                <BrokenView />
            </AppErrorBoundary>,
        );

        expect(screen.getByRole("alert")).toBeInTheDocument();
        expect(
            screen.getByRole("heading", {
                name: "We couldn't display this page.",
            }),
        ).toBeInTheDocument();
        expect(
            screen.getByRole("button", { name: "Try again" }),
        ).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Go home" })).toHaveAttribute(
            "href",
            "/",
        );
        expect(document.body).not.toHaveTextContent("private-expense-token");
        expect(onError).toHaveBeenCalledWith();
    });

    it("retries the failed subtree", () => {
        vi.spyOn(console, "error").mockImplementation(() => undefined);
        let shouldThrow = true;

        function RecoverableView() {
            if (shouldThrow) {
                throw new Error("temporary failure");
            }

            return <p>Page recovered</p>;
        }

        render(
            <AppErrorBoundary>
                <RecoverableView />
            </AppErrorBoundary>,
        );

        shouldThrow = false;
        fireEvent.click(screen.getByRole("button", { name: "Try again" }));

        expect(screen.getByText("Page recovered")).toBeInTheDocument();
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });
});
