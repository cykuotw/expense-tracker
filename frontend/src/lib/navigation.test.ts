import { describe, expect, it } from "vitest";
import { internalDestination, NavigationStore } from "./navigation";

describe("navigation provenance", () => {
    it("returns to the originating monthly review through repeated edit/save round trips", () => {
        const store = new NavigationStore();
        store.observe("group", "/group/g", "POP");
        store.observe("review", "/group/g/monthly-review/2026-08", "PUSH");
        store.observe("expense", "/expense/e", "PUSH");
        for (let attempt = 0; attempt < 3; attempt++) {
            store.observe(`edit-${attempt}`, "/expense/e/edit", "PUSH");
            expect(store.returnAction(`edit-${attempt}`, "/expense/e/edit", "/", "/expense/e").delta).toBe(-1);
            store.observe("expense", "/expense/e", "POP");
        }
        expect(store.target("expense", "/expense/e", "/group/g").url).toBe("/group/g/monthly-review/2026-08");
        expect(store.returnAction("expense", "/expense/e", "/group/g").delta).toBe(-1);
    });

    it("keeps the group as the review return source across month history", () => {
        const store = new NavigationStore();
        store.observe("group", "/group/g", "POP");
        store.observe("august", "/group/g/monthly-review/2026-08", "PUSH");
        store.observe("july", "/group/g/monthly-review/2026-07", "PUSH");
        expect(store.returnAction("july", "/group/g/monthly-review/2026-07", "/group/g").delta).toBe(-2);
        store.observe("august", "/group/g/monthly-review/2026-08", "POP");
        expect(store.target("august", "/group/g/monthly-review/2026-08", "/group/g").url).toBe("/group/g");
    });

    it("separates two entries for the same expense and truncates forward history on new entry", () => {
        const store = new NavigationStore();
        store.observe("group", "/group/g", "POP");
        store.observe("first", "/expense/e", "PUSH");
        store.observe("group", "/group/g", "POP");
        store.observe("review", "/group/g/monthly-review/2026-08", "PUSH");
        store.observe("second", "/expense/e", "PUSH");
        expect(store.target("first", "/expense/e", "/").url).toBe("/group/g");
        expect(store.target("second", "/expense/e", "/").url).toContain("monthly-review");
        expect(store.history).not.toContain("first");
    });

    it("does not infer predecessors from a new tab, reload, or unknown POP entry", () => {
        const store = new NavigationStore();
        store.observe("direct", "/expense/e", "POP");
        expect(store.returnAction("direct", "/expense/e", "/group/g")).toEqual({ url: "/group/g" });
        store.observe("unknown", "/create_expense", "POP");
        expect(store.returnAction("unknown", "/create_expense", "/group/null")).toEqual({ url: "/" });
        store.observe("group", "/group/g", "PUSH");
        store.observe("untracked-expense", "/expense/e", "POP");
        expect(store.returnAction("untracked-expense", "/expense/e", "/group/other")).toEqual({ url: "/group/other" });
    });

    it("uses the saved group on creation and does not return to the creation form from that group", () => {
        const store = new NavigationStore();
        store.observe("home", "/", "POP");
        store.observe("create", "/create_expense", "PUSH");
        expect(store.returnAction("create", "/create_expense", "/", "/group/new")).toEqual({ url: "/group/new" });
        store.observe("new", "/group/new", "REPLACE");
        expect(store.target("new", "/group/new", "/").url).toBe("/");
    });

    it("returns nested steps to the exact parent draft URL even if its selected group changed", () => {
        const store = new NavigationStore();
        store.observe("form", "/create_expense?g=first", "POP");
        store.observe("split", "/create_expense/split?g=second", "PUSH");
        expect(store.returnAction("split", "/create_expense/split?g=second", "/create_expense?g=second")).toEqual({ delta: -1, url: "/create_expense?g=first" });
        store.observe("form", "/create_expense?g=first", "POP");
        store.observe("receipt", "/create_expense/receipt?g=first", "PUSH");
        store.observe("review", "/create_expense/receipt/review?g=first", "PUSH");
        expect(store.returnAction("review", "/create_expense/receipt/review?g=first", "/create_expense?g=first")).toEqual({ delta: -2, url: "/create_expense?g=first" });
    });

    it("bounds retained entries and safely falls back after provenance is evicted", () => {
        const store = new NavigationStore();
        store.observe("home", "/", "POP");
        for (let index = 0; index < 100; index++) store.observe(String(index), `/group/g/monthly-review/2026-${index % 2 ? "08" : "07"}`, "PUSH");
        expect(store.entries.size).toBe(80);
        expect(store.history.length).toBe(80);
        expect(store.returnAction("99", "/group/g/monthly-review/2026-08", "/group/g").delta).toBeUndefined();
    });

    it.each(["https://example.com", "//example.com", "/\\example.com", "/group/null", "/group/undefined", "/group/%2f", "/group/../login", "/login", "/expense/e/other", "/group/g/monthly-review/2026-13"])("rejects invalid return destination %s", (url) => {
        expect(internalDestination(url)).toBeUndefined();
    });
});
