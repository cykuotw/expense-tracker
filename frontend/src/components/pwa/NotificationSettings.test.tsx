import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import NotificationSettings from "./NotificationSettings";

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));

vi.mock("../../lib/api", async () => {
    const actual = await vi.importActual<typeof import("../../lib/api")>("../../lib/api");
    return { ...actual, apiFetch: (...args: unknown[]) => apiFetchMock(...args) };
});

const group = {
    id: "group-1",
    groupName: "Daily expenses",
    description: "",
    currency: "CAD",
    groupType: "home",
};

function response(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

function supportWebPush() {
    Object.defineProperty(window, "isSecureContext", { configurable: true, value: true });
    Object.defineProperty(window, "Notification", {
        configurable: true,
        value: { requestPermission: vi.fn() },
    });
    Object.defineProperty(window, "PushManager", { configurable: true, value: class PushManager {} });
    Object.defineProperty(navigator, "serviceWorker", {
        configurable: true,
        value: { ready: Promise.resolve({ pushManager: { getSubscription: vi.fn() } }) },
    });
}

function mockSettings(muted: boolean) {
    apiFetchMock.mockImplementation((path: string, init?: RequestInit) => {
        if (path === "/notifications/settings") {
            return Promise.resolve(response({
                enabled: true,
                showDetails: false,
                vapidPublicKey: "public-key",
                mutedGroups: muted ? [{ groupId: group.id, groupName: group.groupName, muted: true }] : [],
            }));
        }
        if (path === "/groups") return Promise.resolve(response([group]));
        if (path === `/notifications/groups/${group.id}/mute` && init?.method === "PUT") {
            return Promise.resolve(new Response(null, { status: 204 }));
        }
        return Promise.resolve(response({}, 404));
    });
}

describe("NotificationSettings", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        supportWebPush();
    });

    afterEach(cleanup);

    it("is limited to the mobile layout", () => {
        mockSettings(false);
        render(<NotificationSettings />);

        expect(screen.getByText("Activity notifications").closest("section")).toHaveClass(
            "md:hidden",
        );
    });

    it("presents a muted group as notifications off and enables it with positive semantics", async () => {
        mockSettings(true);
        render(<NotificationSettings />);

        const checkbox = await screen.findByRole("checkbox", { name: /Daily expenses/ });
        expect(screen.getByRole("group", { name: "Group notifications" })).toBeInTheDocument();
        expect(screen.getByText("Notifications off")).toBeInTheDocument();
        expect(checkbox).not.toBeChecked();

        fireEvent.click(checkbox);

        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledWith(
            "/notifications/groups/group-1/mute",
            { method: "PUT", body: JSON.stringify({ muted: false }) },
        ));
    });

    it("presents an active group as notifications on and mutes it when unchecked", async () => {
        mockSettings(false);
        render(<NotificationSettings />);

        const checkbox = await screen.findByRole("checkbox", { name: /Daily expenses/ });
        expect(screen.getByText("Notifications on")).toBeInTheDocument();
        expect(checkbox).toBeChecked();

        fireEvent.click(checkbox);

        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledWith(
            "/notifications/groups/group-1/mute",
            { method: "PUT", body: JSON.stringify({ muted: true }) },
        ));
    });
});
