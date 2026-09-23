import { describe, expect, it } from "vitest";
import { BalanceData } from "../types/balance";
import { buildEstimatedSettlementEntries } from "./settlementPreview";

const balanceData: BalanceData = {
    currency: "CAD",
    currentUser: "current-user",
    balances: [
        {
            id: "cad-tttt",
            senderUserId: "tttt",
            senderUsername: "ttttttttt",
            receiverUserId: "current-user",
            receiverUsername: "You",
            balance: "60",
            currency: "CAD",
        },
        {
            id: "usd-tttt",
            senderUserId: "current-user",
            senderUsername: "You",
            receiverUserId: "tttt",
            receiverUsername: "ttttttttt",
            balance: "30",
            currency: "USD",
        },
        {
            id: "cad-test",
            senderUserId: "test-user",
            senderUsername: "testuser2",
            receiverUserId: "current-user",
            receiverUsername: "You",
            balance: "60",
            currency: "CAD",
        },
        {
            id: "twd-test",
            senderUserId: "current-user",
            senderUsername: "You",
            receiverUserId: "test-user",
            receiverUsername: "testuser2",
            balance: "1000",
            currency: "TWD",
        },
    ],
    settlementPreview: {
        currency: "CAD",
        complete: true,
        netAmount: "36.5",
        contributions: [
            { balanceId: "cad-tttt", sourceCurrency: "CAD", sourceAmount: "60", rate: "1", previewAmount: "60" },
            { balanceId: "usd-tttt", sourceCurrency: "USD", sourceAmount: "-30", rate: "1.35", previewAmount: "-40.50" },
            { balanceId: "cad-test", sourceCurrency: "CAD", sourceAmount: "60", rate: "1", previewAmount: "60" },
            { balanceId: "twd-test", sourceCurrency: "TWD", sourceAmount: "-1000", rate: "0.043", previewAmount: "-43" },
        ],
        missingRateCurrencies: [],
    },
};

describe("buildEstimatedSettlementEntries", () => {
    it("groups exact converted amounts by counterparty", () => {
        expect(buildEstimatedSettlementEntries(balanceData)).toEqual([
            {
                counterpartyUserId: "tttt",
                label: "ttttttttt owes you",
                amount: "19.5",
                currency: "CAD",
                tone: "text-success",
            },
            {
                counterpartyUserId: "test-user",
                label: "testuser2 owes you",
                amount: "17",
                currency: "CAD",
                tone: "text-success",
            },
        ]);
    });

    it("reverses the label and exposes a positive display amount when the net is negative", () => {
        const owing = {
            ...balanceData,
            settlementPreview: {
                ...balanceData.settlementPreview!,
                contributions: [
                    { balanceId: "usd-tttt", sourceCurrency: "USD", sourceAmount: "-30", rate: "1.35", previewAmount: "-40.5" },
                ],
            },
        };

        expect(buildEstimatedSettlementEntries(owing)[0]).toEqual(
            expect.objectContaining({ label: "You owe ttttttttt", amount: "40.5", tone: "text-destructive" })
        );
    });

    it("returns no estimated rows when the preview is incomplete", () => {
        const incomplete = {
            ...balanceData,
            settlementPreview: { ...balanceData.settlementPreview!, complete: false },
        };
        expect(buildEstimatedSettlementEntries(incomplete)).toEqual([]);
    });
});
