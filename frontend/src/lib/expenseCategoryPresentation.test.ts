import { describe, expect, it } from "vitest";
import {
    getExpenseCategoryPresentation,
    getExpenseTypePresentation,
} from "./expenseCategoryPresentation";

describe("getExpenseCategoryPresentation", () => {
    it("uses a consistent category presentation and a safe fallback", () => {
        expect(getExpenseCategoryPresentation("Food and drink").label).toBe(
            "Food and drink"
        );
        expect(getExpenseCategoryPresentation("unknown").label).toBe("Other");
    });

    it("keeps a category colour while assigning unique icons to its types", () => {
        const games = getExpenseTypePresentation("Entertainment", "Games");
        const movies = getExpenseTypePresentation("Entertainment", "Movies");

        expect(games.iconClassName).toBe(movies.iconClassName);
        expect(games.icon).not.toBe(movies.icon);
    });
});
