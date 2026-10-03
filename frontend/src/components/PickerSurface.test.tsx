import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import PickerSurface from "./PickerSurface";

function Picker() {
    const [open, setOpen] = useState(false);
    return <><PickerSurface open={open} onClose={() => setOpen(false)} label="Types"
        trigger={<button type="button" onClick={() => setOpen(!open)}>Choose type</button>}>
        <button type="button" role="option" tabIndex={-1} aria-selected={false}>First</button>
        <button type="button" role="option" tabIndex={-1} aria-selected={true} onClick={() => setOpen(false)}>Last</button>
    </PickerSurface><button type="button">Outside</button></>;
}

afterEach(() => { cleanup(); vi.restoreAllMocks(); });

describe("PickerSurface", () => {
    it("portals options out of the card and lets keyboard focus reach the selected/last option", () => {
        const { container } = render(<Picker />);
        const trigger = screen.getByRole("button", { name: "Choose type" });
        act(() => trigger.focus());
        fireEvent.click(trigger);
        expect(container.querySelector('[role="listbox"]')).toBeNull();
        fireEvent.keyDown(trigger, { key: "ArrowDown" });
        expect(screen.getByRole("option", { name: "Last" })).toHaveFocus();
        fireEvent.keyDown(document.activeElement!, { key: "Home" });
        expect(screen.getByRole("option", { name: "First" })).toHaveFocus();
        fireEvent.keyDown(document.activeElement!, { key: "End" });
        expect(screen.getByRole("option", { name: "Last" })).toHaveFocus();
        fireEvent.keyDown(document.activeElement!, { key: "Escape" });
        expect(trigger).toHaveFocus();
        expect(screen.queryByRole("listbox")).toBeNull();
    });
    it("dismisses on an outside pointer press without trapping focus", () => {
        render(<Picker />);
        fireEvent.click(screen.getByRole("button", { name: "Choose type" }));
        fireEvent.pointerDown(screen.getByRole("button", { name: "Outside" }));
        expect(screen.queryByRole("listbox")).toBeNull();
    });
});
