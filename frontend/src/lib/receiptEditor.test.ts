import { describe, expect, it } from "vitest";

import {
    commitReceiptEdit,
    createReceiptHistory,
    normalizeRect,
    orientedDimensions,
    redoReceiptEdit,
    type ReceiptEditState,
    rotateEditStateClockwise,
    undoReceiptEdit,
    updateRect,
} from "./receiptEditor";

describe("receipt editor geometry", () => {
    it("keeps normalized rectangles inside the image", () => {
        const normalized = normalizeRect({ x: 0.98, y: 0.99, width: 0.2, height: 0.2 });
        expect(normalized.x).toBeCloseTo(0.98);
        expect(normalized.y).toBeCloseTo(0.99);
        expect(normalized.width).toBeCloseTo(0.02);
        expect(normalized.height).toBeCloseTo(0.01);
        expect(updateRect({ x: 0.8, y: 0.7, width: 0.2, height: 0.3 }, { width: 0.5 })).toEqual({
            x: 0.5,
            y: 0.7,
            width: 0.5,
            height: 0.3,
        });
    });

    it("rotates crop and masks with the source pixels", () => {
        const initial = {
            rotation: 0 as const,
            crop: { x: 0.1, y: 0.2, width: 0.5, height: 0.6 },
            masks: [{ id: "private", x: 0.2, y: 0.3, width: 0.1, height: 0.2 }],
        };
        const rotated = rotateEditStateClockwise(initial);
        expect(rotated.rotation).toBe(90);
        expect(rotated.crop.x).toBeCloseTo(0.2);
        expect(rotated.crop.y).toBeCloseTo(0.1);
        expect(rotated.crop.width).toBeCloseTo(0.6);
        expect(rotated.crop.height).toBeCloseTo(0.5);
        expect(rotated.masks[0]?.id).toBe("private");
        expect(rotated.masks[0]?.x).toBeCloseTo(0.5);
        expect(rotated.masks[0]?.y).toBeCloseTo(0.2);
        expect(rotated.masks[0]?.width).toBeCloseTo(0.2);
        expect(rotated.masks[0]?.height).toBeCloseTo(0.1);
        expect(orientedDimensions(1200, 800, rotated.rotation)).toEqual({ width: 800, height: 1200 });

        let fullTurn: ReceiptEditState = initial;
        for (let index = 0; index < 4; index += 1) fullTurn = rotateEditStateClockwise(fullTurn);
        expect(fullTurn.rotation).toBe(initial.rotation);
        expect(fullTurn.crop.x).toBeCloseTo(initial.crop.x);
        expect(fullTurn.crop.y).toBeCloseTo(initial.crop.y);
        expect(fullTurn.crop.width).toBeCloseTo(initial.crop.width);
        expect(fullTurn.crop.height).toBeCloseTo(initial.crop.height);
    });
});

describe("receipt editor history", () => {
    it("records semantic edits and clears redo after a new edit", () => {
        const initial = createReceiptHistory();
        const cropped = {
            ...initial.present,
            crop: { x: 0.1, y: 0.1, width: 0.8, height: 0.8 },
        };
        const committed = commitReceiptEdit(initial, cropped);
        const undone = undoReceiptEdit(committed);
        expect(undone.present).toEqual(initial.present);
        expect(redoReceiptEdit(undone).present).toEqual(cropped);

        const replacement = commitReceiptEdit(undone, {
            ...undone.present,
            rotation: 90,
        });
        expect(replacement.future).toEqual([]);
    });
});
