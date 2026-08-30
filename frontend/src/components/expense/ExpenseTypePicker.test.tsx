import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ExpenseTypePicker } from "./ExpenseTypePicker";

describe("ExpenseTypePicker", () => {
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

        expect(screen.queryByRole("button", { name: "Games" })).toBeNull();

        fireEvent.click(screen.getByRole("button", { name: "Movies" }));

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
});
