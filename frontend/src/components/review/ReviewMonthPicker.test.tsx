import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReviewMonthPicker } from "./ReviewMonthPicker";

describe("ReviewMonthPicker", () => {
    afterEach(cleanup);

    it("selects a month from a custom year menu and closes the menu", () => {
        const onChange = vi.fn();
        render(
            <ReviewMonthPicker
                value="2026-08"
                minimumMonth="2026-01"
                maximumMonth="2028-12"
                onChange={onChange}
            />,
        );

        const trigger = screen.getByRole("button", { name: "Review month" });
        expect(trigger).toHaveAttribute("aria-expanded", "false");
        fireEvent.click(trigger);
        fireEvent.click(screen.getByRole("button", { name: "Review year" }));
        fireEvent.click(screen.getByRole("option", { name: "2027" }));
        fireEvent.click(screen.getByRole("button", { name: "January 2027" }));

        expect(onChange).toHaveBeenCalledWith("2027-01");
        expect(trigger).toHaveAttribute("aria-expanded", "false");
    });

    it("disables months outside the available range", () => {
        render(
            <ReviewMonthPicker
                value="2026-08"
                minimumMonth="2026-03"
                maximumMonth="2026-08"
                onChange={vi.fn()}
            />,
        );

        fireEvent.click(screen.getByRole("button", { name: "Review month" }));

        expect(screen.getByRole("button", { name: "January 2026" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "March 2026" })).toBeEnabled();
        expect(screen.getByRole("button", { name: "September 2026" })).toBeDisabled();
    });

    it("closes on Escape without changing the month", () => {
        const onChange = vi.fn();
        render(<ReviewMonthPicker value="2026-08" onChange={onChange} />);

        fireEvent.click(screen.getByRole("button", { name: "Review month" }));
        fireEvent.keyDown(screen.getByRole("dialog", { name: "Choose review month" }), {
            key: "Escape",
        });

        expect(screen.queryByRole("dialog", { name: "Choose review month" })).toBeNull();
        expect(onChange).not.toHaveBeenCalled();
    });
});
