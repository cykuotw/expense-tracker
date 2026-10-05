import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import ArchivedGroups from "./ArchivedGroups";

const { fetchMock } = vi.hoisted(() => ({ fetchMock: vi.fn() }));
vi.mock("../../lib/api", () => ({ apiFetch: fetchMock, asArray: (value: unknown) => Array.isArray(value) ? value : [], getResponseErrorMessage: async () => "Archive list unavailable" }));
afterEach(() => { cleanup(); fetchMock.mockReset(); });
const page = (groups: { id: string; groupName: string }[], nextCursor = "") => new Response(JSON.stringify({ groups, nextCursor }), { status: 200 });

describe("Archived groups", () => {
    it("loads only on expansion and retains pages across collapse, with ID deduplication", async () => {
        fetchMock.mockResolvedValueOnce(page([{ id: "a", groupName: "Old trip" }], "next"))
            .mockResolvedValueOnce(page([{ id: "a", groupName: "Old trip" }, { id: "b", groupName: "Old event" }]));
        render(<MemoryRouter><ArchivedGroups /></MemoryRouter>);
        expect(fetchMock).not.toHaveBeenCalled();
        const disclosure = screen.getByRole("button", { name: "Archived groups" });
        const content = document.getElementById(disclosure.getAttribute("aria-controls")!);
        expect(content).not.toBeVisible();
        fireEvent.click(disclosure);
        expect(disclosure).toHaveAttribute("aria-expanded", "true");
        expect(content).toBeVisible();
        expect(await screen.findByRole("link", { name: "Old trip, Archived" })).toHaveAttribute("href", "/group/a");
        fireEvent.click(await screen.findByRole("button", { name: "Load more archived groups" }));
        expect(await screen.findByRole("link", { name: "Old event, Archived" })).toBeVisible();
        expect(screen.getAllByRole("link", { name: "Old trip, Archived" })).toHaveLength(1);
        expect(fetchMock.mock.calls[1][0]).toContain("cursor=next");
        fireEvent.click(disclosure);
        expect(disclosure).toHaveAttribute("aria-expanded", "false");
        expect(content).not.toBeVisible();
        fireEvent.click(disclosure);
        expect(screen.getByText("No more archived groups")).toBeVisible();
        expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    it("keeps successful pages and retries the failed cursor", async () => {
        fetchMock.mockResolvedValueOnce(page([{ id: "a", groupName: "Old trip" }], "next"))
            .mockResolvedValueOnce(new Response("{}", { status: 503 }))
            .mockResolvedValueOnce(page([{ id: "b", groupName: "Old event" }]));
        render(<MemoryRouter><ArchivedGroups /></MemoryRouter>);
        fireEvent.click(screen.getByRole("button", { name: "Archived groups" }));
        fireEvent.click(await screen.findByRole("button", { name: "Load more archived groups" }));
        expect(await screen.findByRole("alert")).toHaveTextContent("Archive list unavailable");
        expect(screen.getByRole("link", { name: "Old trip, Archived" })).toBeVisible();
        fireEvent.click(screen.getByRole("button", { name: "Try again" }));
        expect(await screen.findByRole("link", { name: "Old event, Archived" })).toBeVisible();
        expect(fetchMock.mock.calls[2][0]).toEqual(fetchMock.mock.calls[1][0]);
    });

    it("shows a retry after an initial failure and a distinct empty state", async () => {
        fetchMock.mockRejectedValueOnce(new Error("Connection lost")).mockResolvedValueOnce(page([]));
        render(<MemoryRouter><ArchivedGroups /></MemoryRouter>);
        fireEvent.click(screen.getByRole("button", { name: "Archived groups" }));
        expect(await screen.findByRole("alert")).toHaveTextContent("Connection lost");
        fireEvent.click(screen.getByRole("button", { name: "Try again" }));
        await waitFor(() => expect(screen.getByText("No archived groups yet.")).toBeVisible());
    });
});
