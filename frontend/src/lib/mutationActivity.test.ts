import { describe, expect, it, vi } from "vitest";

import {
    beginMutationActivity,
    subscribeMutationActivity,
} from "./mutationActivity";

describe("mutation activity", () => {
    it("remains active until every overlapping mutation finishes", () => {
        const listener = vi.fn();
        const unsubscribe = subscribeMutationActivity(listener);
        const finishFirst = beginMutationActivity();
        const finishSecond = beginMutationActivity();

        finishFirst();
        expect(listener).toHaveBeenLastCalledWith(true);

        finishSecond();
        expect(listener).toHaveBeenLastCalledWith(false);
        unsubscribe();
    });

    it("makes a mutation completion callback idempotent", () => {
        const listener = vi.fn();
        const unsubscribe = subscribeMutationActivity(listener);
        const finish = beginMutationActivity();

        finish();
        finish();

        expect(listener).toHaveBeenLastCalledWith(false);
        unsubscribe();
    });
});
