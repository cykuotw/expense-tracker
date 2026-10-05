import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import NavigationProvider from "../../contexts/NavigationProvider";
import type { GroupInfo } from "../../types/group";
import GroupLifecycleControls from "./GroupLifecycleControls";

const { fetchMock } = vi.hoisted(() => ({ fetchMock: vi.fn() }));
vi.mock("../../lib/api", () => ({ apiFetch: fetchMock, getResponseErrorMessage: async () => "Settle the group first" }));
vi.mock("react-hot-toast", () => ({ toast: { success: vi.fn() } }));
afterEach(() => { cleanup(); fetchMock.mockReset(); });
const group = { isActive: true, canManageLifecycle: true, canArchive: true, canRestore: false } as GroupInfo;
function show(value = group, refresh = vi.fn()) {
    render(<MemoryRouter initialEntries={["/group/g/edit"]}><NavigationProvider><Routes>
        <Route path="/group/:id/edit" element={<GroupLifecycleControls groupId="g" group={value} onRefresh={refresh} />} />
        <Route path="/" element={<h1>Home destination</h1>} />
        <Route path="/group/g" element={<h1>Group destination</h1>} />
    </Routes></NavigationProvider></MemoryRouter>);
}
describe("Group lifecycle", () => {
    it("requires confirmation, supports cancel, and returns Home only after archive succeeds", async () => {
        fetchMock.mockResolvedValue(new Response("{}", { status: 201 }));
        show();
        fireEvent.click(screen.getByRole("button", { name: "Archive group" }));
        expect(screen.getByRole("alertdialog")).toHaveTextContent("history are retained");
        fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
        expect(fetchMock).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole("button", { name: "Archive group" }));
        fireEvent.click(screen.getByRole("button", { name: "Confirm archive" }));
        expect(await screen.findByRole("heading", { name: "Home destination" })).toBeVisible();
        expect(fetchMock).toHaveBeenCalledWith("/archive_group/g", { method: "PUT" });
    });
    it("refreshes eligibility after a conflict and retains a recoverable page", async () => {
        fetchMock.mockResolvedValue(new Response("{}", { status: 409 }));
        const refresh = vi.fn();
        show(group, refresh);
        fireEvent.click(screen.getByRole("button", { name: "Archive group" }));
        fireEvent.click(screen.getByRole("button", { name: "Confirm archive" }));
        expect(await screen.findByRole("alert")).toHaveTextContent("Settle the group first");
        expect(refresh).toHaveBeenCalledOnce();
        expect(screen.getByRole("heading", { name: "Group status" })).toBeVisible();
    });
    it("restores without ordinary edit permission and returns to group detail", async () => {
        fetchMock.mockResolvedValue(new Response("{}", { status: 201 }));
        show({ ...group, isActive: false, canArchive: false, canRestore: true, detailsEditable: false });
        fireEvent.click(screen.getByRole("button", { name: "Restore group" }));
        expect(await screen.findByRole("heading", { name: "Group destination" })).toBeVisible();
        expect(fetchMock).toHaveBeenCalledWith("/restore_group/g", { method: "PUT" });
    });
    it("explains ineligibility and does not expose lifecycle actions to non-creators", async () => {
        show({ ...group, canArchive: false, archiveBlockedReason: "Settle all expenses first" });
        expect(screen.getByRole("button", { name: "Archive group" })).toBeDisabled();
        expect(screen.getByText("Settle all expenses first")).toBeVisible();
        cleanup();
        show({ ...group, canManageLifecycle: false });
        await waitFor(() => expect(screen.queryByRole("heading", { name: "Group status" })).not.toBeInTheDocument());
    });
});
