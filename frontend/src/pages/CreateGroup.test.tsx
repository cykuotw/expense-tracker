import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import CreateGroup from "./CreateGroup";

const { apiFetchMock, toastErrorMock, toastSuccessMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    toastErrorMock: vi.fn(),
    toastSuccessMock: vi.fn(),
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

vi.mock("react-hot-toast", () => ({
    toast: {
        error: toastErrorMock,
        success: toastSuccessMock,
    },
}));

vi.mock("../hooks/useCurrencies", () => ({
    useCurrencies: () => ({
        currencies: [
            {
                code: "CAD",
                displayName: "Canadian Dollar",
                minorUnitDigits: 2,
                amountDigits: 2,
            },
        ],
        loading: false,
        error: null,
        reload: vi.fn(),
    }),
}));

function jsonResponse(body: unknown, status = 200) {
    return new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
    });
}

describe("CreateGroup member management", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation((path: string) => {
            if (path === "/related_member") {
                return Promise.resolve(
                    jsonResponse([
                        {
                            userId: "member-1",
                            username: "Alex Member",
                            email: "alex@example.com",
                            existInGroup: false,
                        },
                    ])
                );
            }
            if (path === "/create_group") {
                return Promise.resolve(
                    jsonResponse({ groupId: "new-group" }, 201)
                );
            }
            if (path === "/related_member?g=new-group") {
                return Promise.resolve(jsonResponse([]));
            }
            throw new Error(`Unexpected path: ${path}`);
        });
    });

    it("creates the group and selected members in one request", async () => {
        render(
            <MemoryRouter>
                <CreateGroup />
            </MemoryRouter>
        );

        expect(
            screen.getByRole("heading", { name: "Manage members", level: 2 })
        ).toBeVisible();
        const member = await screen.findByRole("checkbox", {
            name: /Alex Member\s*alex@example\.com/,
        });
        expect(member).toHaveAttribute("form", "create-group-form");
        fireEvent.click(member);

        fireEvent.change(screen.getByPlaceholderText("Group Name"), {
            target: { value: "Weekend trip" },
        });
        fireEvent.click(screen.getByRole("button", { name: "Create Group" }));

        expect(
            await screen.findByRole("heading", { name: "Manage group members" })
        ).toBeVisible();
        expect(
            screen.getByRole("heading", { name: "Manage members", level: 2 })
        ).toBeVisible();
        expect(
            screen.getAllByRole("link", { name: "View group" })
        ).not.toHaveLength(0);
        expect(
            screen.queryByRole("button", { name: "Create Group" })
        ).not.toBeInTheDocument();
        await waitFor(() => {
            expect(apiFetchMock).toHaveBeenCalledWith(
                "/related_member?g=new-group",
                expect.any(Object)
            );
        });
        expect(await screen.findByText("No friends found")).toBeVisible();
        expect(
            apiFetchMock.mock.calls.filter(([path]) => path === "/create_group")
        ).toHaveLength(1);
        const createCall = apiFetchMock.mock.calls.find(
            ([path]) => path === "/create_group"
        );
        expect(JSON.parse(createCall?.[1]?.body as string)).toMatchObject({
            groupName: "Weekend trip",
            memberIds: ["member-1"],
        });
    });
});
