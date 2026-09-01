import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import MobilePageHeader from "./MobilePageHeader";

describe("MobilePageHeader", () => {
    it("keeps explicit navigation, a centered title, and the primary action together", () => {
        render(
            <MemoryRouter>
                <MobilePageHeader
                    title="Edit group"
                    backTo="/group/group-1"
                    backLabel="Back to group"
                    action={<button type="button" aria-label="Save group">Save</button>}
                />
            </MemoryRouter>,
        );

        const back = screen.getByRole("link", { name: "Back to group" });
        expect(back).toHaveAttribute("href", "/group/group-1");
        expect(back).toHaveClass("ui-button-outline", "mobile-page-header__back");
        expect(screen.getByRole("heading", { name: "Edit group" })).toBeVisible();
        expect(screen.getByRole("button", { name: "Save group" })).toBeVisible();
    });
});
