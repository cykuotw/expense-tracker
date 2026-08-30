import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExpenseTypePicker } from "./ExpenseTypePicker";

describe("ExpenseTypePicker", () => {
    afterEach(cleanup);

    it("filters types and returns the selected expense type ID", () => {
        const onChange = vi.fn();

        render(
            <ExpenseTypePicker
                expenseTypes={[
                    {
                        id: "games",
                        category: "Entertainment",
                        name: "Games",
                    },
                    {
                        id: "movies",
                        category: "Entertainment",
                        name: "Movies",
                    },
                ]}
                value=""
                onChange={onChange}
            />
        );

        const trigger = screen.getByRole("button", {
            name: /choose a category/i,
        });
        fireEvent.click(trigger);
        fireEvent.change(screen.getByPlaceholderText("Search categories"), {
            target: { value: "movie" },
        });

        expect(screen.queryByRole("option", { name: "Games" })).toBeNull();

        fireEvent.click(screen.getByRole("option", { name: "Movies" }));

        expect(onChange).toHaveBeenCalledWith("movies");
        expect(trigger).toHaveAttribute("aria-expanded", "false");
    });

    it("closes after its focus leaves the picker", () => {
        render(
            <>
                <ExpenseTypePicker
                    expenseTypes={[
                        {
                            id: "games",
                            category: "Entertainment",
                            name: "Games",
                        },
                    ]}
                    value="games"
                    onChange={vi.fn()}
                />
                <button type="button">Outside</button>
            </>
        );

        const trigger = screen.getByRole("button", { name: /games/i });
        fireEvent.click(trigger);
        expect(screen.getByPlaceholderText("Search categories")).toBeVisible();

        fireEvent.blur(trigger, {
            relatedTarget: screen.getByRole("button", { name: "Outside" }),
        });

        expect(screen.queryByPlaceholderText("Search categories")).toBeNull();
    });

    it("selects an option on pointer press before Safari can close the picker", () => {
        const onChange = vi.fn();

        render(
            <ExpenseTypePicker
                expenseTypes={[
                    { id: "games", category: "Entertainment", name: "Games" },
                ]}
                value=""
                onChange={onChange}
            />
        );

        fireEvent.click(screen.getByRole("button", { name: /choose a category/i }));
        fireEvent.pointerDown(screen.getByRole("option", { name: "Games" }));

        expect(onChange).toHaveBeenCalledWith("games");
        expect(screen.queryByRole("listbox", { name: "Expense type options" })).toBeNull();
    });

    it("does not focus search until the user chooses to search", () => {
        render(
            <ExpenseTypePicker
                expenseTypes={[
                    { id: "games", category: "Entertainment", name: "Games" },
                ]}
                value=""
                onChange={vi.fn()}
            />
        );

        const trigger = screen.getByRole("button", { name: /choose a category/i });
        trigger.focus();
        fireEvent.click(trigger);

        expect(document.activeElement).toBe(trigger);
    });
});
