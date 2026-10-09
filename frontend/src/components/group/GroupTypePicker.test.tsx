import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GroupTypePicker } from "./GroupTypePicker";

describe("GroupTypePicker", () => {
    afterEach(() => {
        cleanup();
        vi.unstubAllGlobals();
    });

    it("reports a touch-safe pointer selection before focus can move", () => {
        const onChange = vi.fn();
        render(<GroupTypePicker value="home" onChange={onChange} />);

        const trigger = screen.getByRole("button", { name: "Group type" });
        expect(trigger).toHaveTextContent("Home");
        fireEvent.click(trigger);
        const trip = screen.getByRole("option", { name: "Trip" });
        fireEvent.pointerDown(trip, { clientY: 100 });
        fireEvent.pointerUp(trip, { clientY: 100 });

        expect(onChange).toHaveBeenCalledWith("trip");
        expect(screen.queryByRole("listbox", { name: "Group type options" })).toBeNull();
    });

    it("keeps the first option visible and applies a selection in one step", () => {
        const onChange = vi.fn();
        render(<GroupTypePicker value="home" onChange={onChange} />);

        fireEvent.click(screen.getByRole("button", { name: "Group type" }));
        const options = screen.getAllByRole("option");
        expect(options[0]).toHaveTextContent("Trip");

        fireEvent.click(options[0]);

        expect(onChange).toHaveBeenCalledWith("trip");
        expect(screen.queryByRole("listbox", { name: "Group type options" })).toBeNull();
    });
});
