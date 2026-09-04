import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { mdiAccountGroupOutline, mdiAirplane } from "@mdi/js";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExpenseFormPicker } from "./ExpenseFormPicker";

describe("ExpenseFormPicker", () => {
    afterEach(cleanup);

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

    it("prefers an option icon over the fallback icon", () => {
        const { container } = render(
            <ExpenseFormPicker
                label="Group"
                emptyLabel="Choose a group"
                icon={mdiAccountGroupOutline}
                value="group-1"
                onChange={vi.fn()}
                options={[
                    { value: "group-1", label: "Trip", icon: mdiAirplane },
                ]}
            />
        );

        expect(container.querySelector("button path")).toHaveAttribute("d", mdiAirplane);
    });

    it("uses an option's semantic icon colour in the trigger and menu", () => {
        render(
            <ExpenseFormPicker
                label="Group"
                emptyLabel="Choose a group"
                value="group-1"
                onChange={vi.fn()}
                options={[
                    {
                        value: "group-1",
                        label: "Trip",
                        icon: mdiAirplane,
                        iconClassName: "bg-rose-100 text-rose-700",
                    },
                    {
                        value: "group-2",
                        label: "Home",
                        icon: mdiAccountGroupOutline,
                        iconClassName: "bg-amber-100 text-amber-700",
                    },
                ]}
            />
        );

        const trigger = screen.getByRole("button", { name: "Group" });
        expect(trigger.querySelector(".bg-rose-100")).toBeInTheDocument();

        fireEvent.click(trigger);
        expect(
            screen
                .getByRole("option", { name: "Home" })
                .querySelector(".bg-amber-100")
        ).toBeInTheDocument();
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

    it("commits a pointer selection when the pointer is released", () => {
        const onChange = vi.fn();

        render(
            <ExpenseFormPicker
                label="Currency"
                emptyLabel="Choose currency"
                value="CAD"
                onChange={onChange}
                options={[
                    { value: "CAD", label: "CAD" },
                    { value: "USD", label: "USD" },
                ]}
            />
        );

        fireEvent.click(screen.getByRole("button", { name: "Currency" }));
        const option = screen.getByRole("option", { name: "USD" });
        fireEvent.pointerDown(option, { clientY: 100 });
        fireEvent.pointerUp(option, { clientY: 100 });

        expect(onChange).toHaveBeenCalledWith("USD");
        expect(screen.queryByRole("listbox", { name: "Currency options" })).toBeNull();
    });

    it("does not select an option while the option list is being scrolled", () => {
        const onChange = vi.fn();

        render(
            <ExpenseFormPicker
                label="Split rule"
                emptyLabel="Choose a split rule"
                mobileMenuPlacement="above"
                value="Equally"
                onChange={onChange}
                options={[
                    { value: "Equally", label: "Equally" },
                    { value: "Unequally", label: "Unequally" },
                ]}
            />
        );

        fireEvent.click(screen.getByRole("button", { name: "Split rule" }));
        expect(screen.getByRole("listbox", { name: "Split rule options" })).toHaveClass(
            "z-[60]",
            "max-h-40",
            "md:max-h-80",
            "overflow-y-auto",
            "bottom-full",
            "md:top-full"
        );
        expect(screen.getByRole("option", { name: "Unequally" })).toHaveClass("min-h-12");
        const option = screen.getByRole("option", { name: "Unequally" });
        fireEvent.pointerDown(option, { clientY: 100 });
        fireEvent.pointerMove(option, { clientY: 120 });
        fireEvent.pointerUp(option, { clientY: 120 });

        expect(onChange).not.toHaveBeenCalled();
        expect(screen.getByRole("listbox", { name: "Split rule options" })).toBeVisible();
    });
});
