import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CurrencyPicker } from "./CurrencyPicker";

const currencies = [
    { code: "AUD", displayName: "Australian Dollar", minorUnitDigits: 2, amountDigits: 2 },
    { code: "CAD", displayName: "Canadian Dollar", minorUnitDigits: 2, amountDigits: 2 },
    { code: "TWD", displayName: "New Taiwan Dollar", minorUnitDigits: 2, amountDigits: 0 },
    { code: "USD", displayName: "US Dollar", minorUnitDigits: 2, amountDigits: 2 },
];

describe("CurrencyPicker", () => {
    afterEach(cleanup);

    it("uses the group-type picker interaction pattern without an ambiguous icon", () => {
        const onChange = vi.fn();
        render(<CurrencyPicker value="CAD" currencies={currencies} onChange={onChange} />);

        const trigger = screen.getByRole("button", { name: "Currency" });
        expect(trigger).toHaveTextContent("CAD — Canadian Dollar");
        fireEvent.click(trigger);

        const options = screen.getAllByRole("option");
        expect(options[0]).toHaveTextContent("AUD — Australian Dollar");
        expect(options[1]).toHaveTextContent("CAD — Canadian Dollar");
        expect(options[2]).toHaveTextContent("TWD — New Taiwan Dollar");

        const option = screen.getByRole("option", { name: "TWD — New Taiwan Dollar" });
        fireEvent.click(option);

        expect(onChange).toHaveBeenCalledWith("TWD");
        expect(screen.queryByRole("listbox", { name: "Currency options" })).toBeNull();
    });

    it("separates the leading main currencies from the remaining API-ordered options", () => {
        render(
            <CurrencyPicker
                value="CAD"
                currencies={[
                    currencies[1],
                    currencies[2],
                    currencies[3],
                    currencies[0],
                ]}
                onChange={vi.fn()}
            />
        );

        fireEvent.click(screen.getByRole("button", { name: "Currency" }));

        expect(screen.getByRole("separator", { name: "Other currencies" })).toBeVisible();
        const options = screen.getAllByRole("option");
        expect(options[3]).toHaveTextContent("AUD — Australian Dollar");
    });
});
