import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import ExpenseSubmitButton from "./ExpenseSubmitButton";

describe("ExpenseSubmitButton", () => {
    afterEach(cleanup);

    it("stays present, disabled, and busy while saving", () => {
        render(
            <ExpenseSubmitButton
                className="ui-button"
                disabled={false}
                idleLabel="Save changes"
                saving
                savingLabel="Saving changes"
            />,
        );

        const button = screen.getByRole("button", { name: "Saving changes" });
        expect(button).toBeDisabled();
        expect(button).toHaveAttribute("aria-busy", "true");
        expect(button).toHaveTextContent("Saving changes");
    });

    it("keeps an icon-only action labeled when idle", () => {
        render(
            <ExpenseSubmitButton
                className="ui-button"
                disabled={false}
                iconOnly
                idleLabel="Save expense"
                saving={false}
                savingLabel="Saving expense"
            />,
        );

        expect(
            screen.getByRole("button", { name: "Save expense" }),
        ).toBeEnabled();
    });
});
