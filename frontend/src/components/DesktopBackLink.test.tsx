import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import DesktopBackLink from "./DesktopBackLink";

describe("DesktopBackLink", () => {
    it("renders a compact semantic navigation link", () => {
        render(
            <MemoryRouter>
                <DesktopBackLink to="/group/group-1" label="Back to group" />
            </MemoryRouter>,
        );

        const link = screen.getByRole("link", { name: "Back to group" });
        expect(link).toHaveAttribute("href", "/group/group-1");
        expect(link).toHaveClass("desktop-back-link");
        expect(link.parentElement).toHaveClass("desktop-page-utility");
        const icon = link.querySelector("span");
        expect(icon).toHaveClass("desktop-back-link__icon");
        expect(icon).toHaveAttribute("aria-hidden", "true");
    });
});
