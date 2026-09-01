import type { ReactNode } from "react";
import { cleanup, render, screen } from "@testing-library/react";
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
        apiFetchMock.mockResolvedValue(
            new Response(
                JSON.stringify({
                    groupName: "Trip",
                    description: "",
                    currency: "CAD",
                    groupType: "trip",
                }),
                {
                    status: 200,
                    headers: { "Content-Type": "application/json" },
                }
            )
        );
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
});
