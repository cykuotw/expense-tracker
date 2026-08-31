import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GroupTypePicker } from "./GroupTypePicker";

describe("GroupTypePicker", () => {
    afterEach(cleanup);

    it("uses the same semantic colour for the selected type and its options", () => {
        render(<GroupTypePicker value="friends" onChange={vi.fn()} />);

        const trigger = screen.getByRole("button", { name: "Group type" });
        expect(trigger).toHaveTextContent("Friends");
        expect(trigger.querySelector(".bg-violet-100")).toBeInTheDocument();

        fireEvent.click(trigger);

        const home = screen.getByRole("option", { name: "Home" });
        expect(home.querySelector(".bg-amber-100")).toBeInTheDocument();
    });

    it("reports a touch-safe pointer selection before focus can move", () => {
        const onChange = vi.fn();
        render(<GroupTypePicker value="home" onChange={onChange} />);

        fireEvent.click(screen.getByRole("button", { name: "Group type" }));
        fireEvent.pointerDown(screen.getByRole("option", { name: "Trip" }));

        expect(onChange).toHaveBeenCalledWith("trip");
        expect(screen.queryByRole("listbox", { name: "Group type options" })).toBeNull();
    });
});
