import { describe, expect, it } from "vitest";
import { getExpenseSubmissionErrorMessage } from "./expenseSubmissionError";

function response(
    body: string,
    contentType: string,
    status = 400,
) {
    return new Response(body, {
        status,
        headers: { "Content-Type": contentType },
    });
}

describe("getExpenseSubmissionErrorMessage", () => {
    it("maps a stable code without exposing the backend message", async () => {
        const message = await getExpenseSubmissionErrorMessage(
            response(
                JSON.stringify({
                    code: "invalid_money",
                    error: "database-password-secret",
                }),
                "application/json",
            ),
            "create",
        );

        expect(message).toBe(
            "Check the expense amount and split, then try again.",
        );
        expect(message).not.toContain("database-password-secret");
    });

    it.each([
        [
            "unknown JSON code",
            JSON.stringify({ code: "unexpected", error: "raw" }),
            "application/json",
        ],
        ["malformed JSON", "{not-json", "application/json"],
        ["plain text", "database-password-secret", "text/plain"],
    ])("uses the safe fallback for %s", async (_name, body, contentType) => {
        await expect(
            getExpenseSubmissionErrorMessage(
                response(body, contentType),
                "update",
            ),
        ).resolves.toBe(
            "We couldn't save your changes. Check your connection and try again.",
        );
    });
});
