import type { ReactNode } from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import EditGroup from "./EditGroup";

const { apiFetchMock, scrollIntoViewMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
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

    afterEach(cleanup);

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

    it("saves currency through the dedicated endpoint", async () => {
        apiFetchMock.mockImplementation((path: string) => Promise.resolve(new Response(
            JSON.stringify(path === "/currencies" ? [
                { code: "CAD", displayName: "Canadian Dollar", minorUnitDigits: 2, amountDigits: 2 },
                { code: "USD", displayName: "US Dollar", minorUnitDigits: 2, amountDigits: 2 },
            ] : path === "/group/group-1" ? {
                groupName: "Trip",
                description: "",
                currency: "CAD",
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

        await waitFor(() => expect(screen.getByRole("button", { name: "Save currency" })).toBeDisabled());
        fireEvent.click(screen.getByRole("button", { name: "Currency" }));
        fireEvent.click(screen.getByRole("option", { name: /USD — US Dollar/ }));
        fireEvent.click(screen.getByRole("button", { name: "Save currency" }));

        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledWith(
            "/group/group-1/currency",
            expect.objectContaining({ method: "PUT", body: JSON.stringify({ currency: "USD" }) }),
        ));
        expect(apiFetchMock).not.toHaveBeenCalledWith(
            "/group/group-1",
            expect.objectContaining({ method: "PUT" }),
        );
    });
});
