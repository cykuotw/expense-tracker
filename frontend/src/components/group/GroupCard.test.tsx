import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import GroupCard from "./GroupCard";
import type { GroupCardData } from "../../types/group";

afterEach(cleanup);
const group: GroupCardData = { id: "trip", groupName: "Completed trip", description: "", currency: "CAD", groupType: "trip", balanceStatus: "settled", balanceAmount: "0" };
describe("GroupCard archive guidance", () => {
    it("does not infer eligibility from a zero personal balance", () => {
        render(<MemoryRouter><GroupCard {...group} /></MemoryRouter>);
        expect(screen.getByText("Your net balance: 0")).toBeVisible();
        expect(screen.queryByText("Inactive for 90+ days")).toBeNull();
    });
    it("offers review and dismissal as separate controls outside the group link", () => {
        render(<MemoryRouter><GroupCard {...group} archiveSuggested /></MemoryRouter>);
        expect(screen.getByText("Inactive for 90+ days")).toBeVisible();
        const review = screen.getByRole("link", { name: "Archive" });
        expect(review).toHaveAttribute("href", "/group/trip/edit#status");
        expect(review.parentElement?.closest("a")).toBeNull();
        const dismiss = screen.getByRole("button", { name: "Dismiss archive suggestion for Completed trip" });
        expect(dismiss.closest("a")).toBeNull();
        fireEvent.click(dismiss);
        expect(screen.queryByText("Inactive for 90+ days")).toBeNull();
        expect(screen.getByRole("link", { name: /Completed trip/ })).toHaveAttribute("href", "/group/trip");
        expect(screen.getByRole("link", { name: /Completed trip/ })).toHaveFocus();
    });
    it("identifies zero converted balances as estimates", () => {
        render(<MemoryRouter><GroupCard {...group} usesSettlementPreview /></MemoryRouter>);
        expect(screen.getByText("Estimated net balance: 0")).toBeVisible();
    });
});
