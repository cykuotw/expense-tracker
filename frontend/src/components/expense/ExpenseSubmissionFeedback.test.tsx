import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import ExpenseSubmissionFeedback from "./ExpenseSubmissionFeedback";

describe("ExpenseSubmissionFeedback", () => {
    afterEach(cleanup);

    it("focuses a persistent alert and retries through the form", async () => {
        const onSubmit = vi.fn();
        render(
            <form
                onSubmit={(event) => {
                    event.preventDefault();
                    onSubmit();
                }}
            >
                <ExpenseSubmissionFeedback
                    error="Check your connection and try again."
                    errorTitle="We couldn't save this expense"
                    idPrefix="create-expense"
                    retryDisabled={false}
                    saving={false}
                    savingLabel="Saving expense"
                />
            </form>,
        );

        const alert = screen.getByRole("alert");
        await waitFor(() => expect(alert).toHaveFocus());
        expect(alert).toHaveAccessibleName("We couldn't save this expense");

        fireEvent.click(screen.getByRole("button", { name: "Try again" }));
        expect(onSubmit).toHaveBeenCalledOnce();
    });

    it("exposes exactly one live saving status", () => {
        render(
            <ExpenseSubmissionFeedback
                error={null}
                errorTitle="We couldn't save this expense"
                idPrefix="create-expense"
                retryDisabled={false}
                saving
                savingLabel="Saving expense"
            />,
        );

        expect(screen.getAllByRole("status")).toHaveLength(1);
        expect(screen.getByRole("status")).toHaveTextContent("Saving expense");
    });
});
