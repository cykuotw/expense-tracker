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

function supportWebPush(subscription: { endpoint: string; unsubscribe?: () => Promise<boolean> } | null = null) {
    Object.defineProperty(window, "isSecureContext", { configurable: true, value: true });
    Object.defineProperty(window, "Notification", {
        configurable: true,
        value: { requestPermission: vi.fn() },
    });
    Object.defineProperty(window, "PushManager", { configurable: true, value: class PushManager {} });
    Object.defineProperty(navigator, "serviceWorker", {
        configurable: true,
        value: { ready: Promise.resolve({ pushManager: { getSubscription: vi.fn().mockResolvedValue(subscription) } }) },
    });
}

function mockSettings(
    muted: boolean,
    options: { subscriptionCount?: number; currentRegistered?: boolean; currentShowDetails?: boolean } = {},
) {
    apiFetchMock.mockImplementation((path: string, init?: RequestInit) => {
        if (path === "/notifications/settings") {
            return Promise.resolve(response({
                enabled: true,
                showDetails: false,
                subscriptionCount: options.subscriptionCount ?? 1,
                vapidPublicKey: "public-key",
                mutedGroups: muted ? [{ groupId: group.id, groupName: group.groupName, muted: true }] : [],
            }));
        }
        if (path === "/notifications/subscriptions/status" && init?.method === "POST") {
            return Promise.resolve(response({
                registered: options.currentRegistered === true,
                showDetails: options.currentShowDetails === true,
            }));
        }
        if (path === "/notifications/subscriptions" && init?.method === "DELETE") {
            return Promise.resolve(new Response(null, { status: 204 }));
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

    it("does not treat another browser's subscription as enabled in this browser", async () => {
        mockSettings(false, { subscriptionCount: 1 });
        render(<NotificationSettings />);

        expect(await screen.findByRole("button", { name: "Enable notifications in this browser" })).toBeInTheDocument();
        expect(screen.getByText("Notifications remain enabled in 1 other browser or device.")).toBeInTheDocument();
    });

    it("uses the current browser subscription for enabled and preview-detail state", async () => {
        supportWebPush({ endpoint: "https://fcm.googleapis.com/fcm/send/current" });
        mockSettings(false, { subscriptionCount: 2, currentRegistered: true, currentShowDetails: true });
        render(<NotificationSettings />);

        expect(await screen.findByRole("button", { name: "Disable notifications in this browser" })).toBeInTheDocument();
        expect(screen.getByRole("checkbox", { name: /Show notification details in this browser/ })).toBeChecked();
        expect(apiFetchMock).toHaveBeenCalledWith(
            "/notifications/subscriptions/status",
            { method: "POST", body: JSON.stringify({ endpoint: "https://fcm.googleapis.com/fcm/send/current" }) },
        );
    });

    it("offers to repair a browser subscription that is missing from the account", async () => {
        supportWebPush({ endpoint: "https://fcm.googleapis.com/fcm/send/stale" });
        mockSettings(false, { subscriptionCount: 0, currentRegistered: false });
        render(<NotificationSettings />);

        expect(await screen.findByRole("button", { name: "Enable notifications in this browser" })).toBeInTheDocument();
        expect(screen.queryByText(/Notifications remain enabled/)).not.toBeInTheDocument();
    });

    it("disables only the current browser endpoint", async () => {
        const unsubscribe = vi.fn().mockResolvedValue(true);
        supportWebPush({ endpoint: "https://fcm.googleapis.com/fcm/send/current", unsubscribe });
        mockSettings(false, { subscriptionCount: 2, currentRegistered: true });
        render(<NotificationSettings />);

        fireEvent.click(await screen.findByRole("button", { name: "Disable notifications in this browser" }));

        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledWith(
            "/notifications/subscriptions",
            { method: "DELETE", body: JSON.stringify({ endpoint: "https://fcm.googleapis.com/fcm/send/current" }) },
        ));
        expect(unsubscribe).toHaveBeenCalledOnce();
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
