import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GroupTypePicker } from "./GroupTypePicker";

describe("GroupTypePicker", () => {
    afterEach(() => {
        cleanup();
        vi.unstubAllGlobals();
    });

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

    it("previews a mobile selection and only applies it after confirmation", () => {
        vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({
            matches: true,
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
            addListener: vi.fn(),
            removeListener: vi.fn(),
        }));
        const onChange = vi.fn();
        render(<GroupTypePicker value="home" onChange={onChange} />);

        fireEvent.click(screen.getByRole("button", { name: "Group type" }));
        fireEvent.click(screen.getByRole("radio", { name: "Trip" }));

        expect(onChange).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Use Trip" })).toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "Use Trip" }));

        expect(onChange).toHaveBeenCalledWith("trip");
        expect(screen.queryByRole("dialog")).toBeNull();
    });

    it("discards a mobile preview when cancelled", () => {
        vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({
            matches: true,
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
            addListener: vi.fn(),
            removeListener: vi.fn(),
        }));
        const onChange = vi.fn();
        render(<GroupTypePicker value="home" onChange={onChange} />);

        fireEvent.click(screen.getByRole("button", { name: "Group type" }));
        fireEvent.click(screen.getByRole("radio", { name: "Trip" }));
        fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

        expect(onChange).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Group type" })).toHaveTextContent("Home");
        expect(screen.queryByRole("dialog")).toBeNull();
    });
});
