import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GroupMemberManager } from "./GroupMemberManager";

const { useAddMemberMock } = vi.hoisted(() => ({
    useAddMemberMock: vi.fn(),
}));

vi.mock("../../hooks/AddMemberContextHooks", () => ({
    useAddMember: useAddMemberMock,
}));

describe("GroupMemberManager", () => {
    it("shows each member display name and readable email in one selection row", () => {
        useAddMemberMock.mockReturnValue({
            loading: false,
            relatedUserList: [
                {
                    userId: "member-1",
                    username: "Readable Name",
                    email: "a.very.long.member.email@example.test",
                    existInGroup: true,
                },
            ],
            email: "",
            setEmail: vi.fn(),
            newMember: null,
            handleSubmitRelatedUsers: vi.fn(),
            handleAddNewMember: vi.fn(),
        });

        render(<GroupMemberManager />);

        const checkbox = screen.getByRole("checkbox", {
            name: /Readable Name\s*a\.very\.long\.member\.email@example\.test/,
        });
        const row = checkbox.closest("label");

        expect(row).not.toBeNull();
        expect(within(row!).getByText("Readable Name")).toBeVisible();
        expect(
            within(row!).getByText("a.very.long.member.email@example.test")
        ).toHaveClass("[overflow-wrap:anywhere]");
    });

});
