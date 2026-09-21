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

    it("associates pre-create selections with the create-group form", () => {
        useAddMemberMock.mockReturnValue({
            loading: false,
            relatedUserList: [
                {
                    userId: "member-1",
                    username: "Readable Name",
                    email: "member@example.test",
                    existInGroup: false,
                },
            ],
            email: "",
            setEmail: vi.fn(),
            newMember: null,
            handleSubmitRelatedUsers: vi.fn(),
            handleAddNewMember: vi.fn(),
        });

        const { container } = render(<GroupMemberManager creationMode />);

        expect(within(container).getByRole("checkbox")).toHaveAttribute(
            "form",
            "create-group-form"
        );
        expect(
            within(container).queryByRole("button", { name: "Update members" })
        ).not.toBeInTheDocument();
        expect(
            within(container).getByText(
                "Selected members will be added when you create the group."
            )
        ).toBeVisible();
    });
});
