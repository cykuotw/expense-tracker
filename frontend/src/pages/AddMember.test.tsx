import type { ReactNode } from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import AddMember from "./AddMember";

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));
vi.mock("../lib/api", async () => ({ ...await vi.importActual("../lib/api"), apiFetch: (...args: unknown[]) => apiFetchMock(...args) }));
vi.mock("../contexts/AddMemberContext", () => ({ AddMemberProvider: ({ children }: { children: ReactNode }) => children }));
vi.mock("../hooks/AddMemberContextHooks", () => ({ useAddMember: () => ({ groupId: "g", loading: false }) }));
vi.mock("../components/group/GroupMemberManager", () => ({ GroupMemberManager: () => <form aria-label="Member editor" /> }));

afterEach(() => { cleanup(); apiFetchMock.mockReset(); });
describe("AddMember lifecycle boundary", () => {
    it("withholds editing while status is unknown and retries a failed lookup", async () => {
        apiFetchMock.mockRejectedValueOnce(new Error("Network unavailable"));
        render(<MemoryRouter><AddMember /></MemoryRouter>);
        expect(screen.queryByRole("form")).toBeNull();
        expect(await screen.findByRole("alert")).toHaveTextContent("Network unavailable");
        apiFetchMock.mockResolvedValue(new Response(JSON.stringify({ isActive: false }), { status: 200 }));
        fireEvent.click(screen.getByRole("button", { name: "Try again" }));
        expect(await screen.findByRole("heading", { name: "This group is archived" })).toBeVisible();
        expect(screen.queryByRole("form")).toBeNull();
        expect(screen.getByRole("link", { name: "View group" })).toHaveAttribute("href", "/group/g");
    });
});
