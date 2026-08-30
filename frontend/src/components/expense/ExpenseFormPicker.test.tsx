import { fireEvent, render, screen } from "@testing-library/react";
import { mdiAccountGroupOutline } from "@mdi/js";
import { describe, expect, it, vi } from "vitest";
import { ExpenseFormPicker } from "./ExpenseFormPicker";

describe("ExpenseFormPicker", () => {
    it("shows the selected option and reports changes", () => {
        const onChange = vi.fn();

        render(
            <ExpenseFormPicker
                label="Group"
                emptyLabel="Choose a group"
                icon={mdiAccountGroupOutline}
                value="group-1"
                onChange={onChange}
                options={[
                    { value: "group-1", label: "Trip" },
                    { value: "group-2", label: "Home" },
                ]}
            />
        );

        const trigger = screen.getByRole("button", { name: "Group" });
        expect(trigger).toHaveTextContent("Trip");

        fireEvent.click(trigger);
        fireEvent.click(screen.getByRole("option", { name: "Home" }));

        expect(onChange).toHaveBeenCalledWith("group-2");
        expect(trigger).toHaveAttribute("aria-expanded", "false");
    });

    it("closes when focus moves outside the picker", () => {
        render(
            <>
                <ExpenseFormPicker
                    label="Currency"
                    emptyLabel="Choose currency"
                    value="CAD"
                    onChange={vi.fn()}
                    options={[{ value: "CAD", label: "CAD" }]}
                />
                <button type="button">Outside</button>
            </>
        );

        const trigger = screen.getByRole("button", { name: "Currency" });
        fireEvent.click(trigger);
        expect(screen.getByRole("listbox", { name: "Currency options" })).toBeVisible();

        fireEvent.blur(trigger, {
            relatedTarget: screen.getByRole("button", { name: "Outside" }),
        });

        expect(screen.queryByRole("listbox", { name: "Currency options" })).toBeNull();
    });
});
