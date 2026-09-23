import type { ReactNode } from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import EditGroup from "./EditGroup";

const { apiFetchMock, providerFetchMock, scrollIntoViewMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    providerFetchMock: vi.fn(),
    scrollIntoViewMock: vi.fn(),
}));

vi.mock("../lib/api", async () => {
    const actual = await vi.importActual<typeof import("../lib/api")>(
        "../lib/api"
    );
    return {
        ...actual,
        apiFetch: (...args: unknown[]) => apiFetchMock(...args),
    };
});

vi.mock("../contexts/AddMemberContext", () => ({
    AddMemberProvider: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("../components/group/GroupMemberManager", () => ({
    GroupMemberManager: () => <div>Member editor</div>,
}));

vi.mock("../components/group/GroupTypePicker", () => ({
    GroupTypePicker: () => <div>Group type picker</div>,
}));

describe("EditGroup member anchor", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        Object.defineProperty(Element.prototype, "scrollIntoView", {
            configurable: true,
            value: scrollIntoViewMock,
        });
        providerFetchMock.mockResolvedValue(new Response(JSON.stringify([
            { date: "2026-09-22", base: "CAD", quote: "USD", rate: 0.7407407407407407 },
        ]), { status: 200, headers: { "Content-Type": "application/json" } }));
        vi.stubGlobal("fetch", providerFetchMock);
        apiFetchMock.mockImplementation((path: string) => Promise.resolve(new Response(
            JSON.stringify(path === "/currencies" ? [
                { code: "CAD", displayName: "Canadian Dollar", minorUnitDigits: 2, amountDigits: 2 },
                { code: "USD", displayName: "US Dollar", minorUnitDigits: 2, amountDigits: 2 },
            ] : {
                groupName: "Trip",
                description: "",
                currency: "CAD",
                groupType: "trip",
                currencyEditable: false,
                detailsEditable: true,
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
        )));
    });

    afterEach(() => {
        cleanup();
        vi.unstubAllGlobals();
    });

    it("scrolls member management into view when opened from the group summary", () => {
        render(
            <MemoryRouter initialEntries={["/group/group-1/edit#members"]}>
                <Routes>
                    <Route path="/group/:id/edit" element={<EditGroup />} />
                </Routes>
            </MemoryRouter>
        );

        expect(screen.getByRole("heading", { name: "Manage members" })).toBeVisible();
        expect(scrollIntoViewMock).toHaveBeenCalledWith({ block: "start" });
    });

    it("enables save only when the loaded group details change", async () => {
        render(
            <MemoryRouter initialEntries={["/group/group-1/edit"]}>
                <Routes>
                    <Route path="/group/:id/edit" element={<EditGroup />} />
                </Routes>
            </MemoryRouter>
        );

        await waitFor(() => {
            expect(screen.getByDisplayValue("Trip")).toBeVisible();
        });
        const saveButtons = screen.getAllByRole("button", { name: "Save group" });
        saveButtons.forEach((button) => expect(button).toBeDisabled());

        fireEvent.change(screen.getByDisplayValue("Trip"), {
            target: { value: "Weekend trip" },
        });
        saveButtons.forEach((button) => expect(button).toBeEnabled());

        fireEvent.change(screen.getByDisplayValue("Weekend trip"), {
            target: { value: "Trip" },
        });
        saveButtons.forEach((button) => expect(button).toBeDisabled());
    });

    it("keeps the group type picker above the currency card", () => {
        render(
            <MemoryRouter initialEntries={["/group/group-1/edit"]}>
                <Routes>
                    <Route path="/group/:id/edit" element={<EditGroup />} />
                </Routes>
            </MemoryRouter>
        );

        expect(document.getElementById("edit-group-form")).toHaveClass(
            "relative",
            "z-10",
        );
    });

    it("saves currency through the dedicated endpoint", async () => {
        apiFetchMock.mockImplementation((path: string) => Promise.resolve(new Response(
            JSON.stringify(path === "/currencies" ? [
                { code: "CAD", displayName: "Canadian Dollar", minorUnitDigits: 2, amountDigits: 2 },
                { code: "USD", displayName: "US Dollar", minorUnitDigits: 2, amountDigits: 2 },
            ] : path === "/group/group-1" ? {
                groupName: "Trip",
                description: "",
                currency: "CAD",
                currencySettings: {
                    settlementPreviewCurrency: "CAD",
                    currencies: [{
                        currency: "CAD",
                        enabledForNewExpenses: true,
                        historical: false,
                        hasCurrentBalance: false,
                        previewRate: "1",
                    }],
                },
                groupType: "trip",
                currencyEditable: true,
                detailsEditable: true,
            } : {}),
            { status: 200, headers: { "Content-Type": "application/json" } },
        )));
        render(
            <MemoryRouter initialEntries={["/group/group-1/edit"]}>
                <Routes>
                    <Route path="/group/:id/edit" element={<EditGroup />} />
                </Routes>
            </MemoryRouter>
        );

        const saveButton = await screen.findByRole("button", { name: "Save currency settings" });
        expect(saveButton).toBeDisabled();
        const addButton = screen.getByRole("button", { name: "Add currency" });
        expect(addButton).toHaveAttribute("data-variant", "default");
        expect(addButton).toHaveClass(
            "min-h-12",
            "w-full",
            "whitespace-nowrap",
            "md:w-auto",
            "md:min-w-36",
            "md:justify-self-end",
        );
        await waitFor(() => expect(addButton).toBeEnabled());
        fireEvent.click(addButton);
        fireEvent.change(await screen.findByLabelText(/1 USD = … CAD/), {
            target: { value: "1.35" },
        });
        fireEvent.click(saveButton);

        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledWith(
            "/group/group-1/currency-settings",
            expect.objectContaining({
                method: "PUT",
                body: JSON.stringify({
                    settlementPreviewCurrency: "CAD",
                    currencies: [
                        { currency: "CAD", enabledForNewExpenses: true, historical: false, hasCurrentBalance: false, previewRate: "1" },
                        { currency: "USD", enabledForNewExpenses: true, historical: false, hasCurrentBalance: false, previewRate: "1.35" },
                    ],
                }),
            }),
        ));
        expect(apiFetchMock).not.toHaveBeenCalledWith(
            "/group/group-1",
            expect.objectContaining({ method: "PUT" }),
        );
    });
});
