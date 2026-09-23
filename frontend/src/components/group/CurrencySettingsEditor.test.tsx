import { useState } from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CurrencyMetadata } from "../../lib/money";
import { GroupCurrencySettings } from "../../types/group";
import { CurrencySettingsEditor } from "./CurrencySettingsEditor";

const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }));

vi.mock("../../lib/api", async () => {
    const actual = await vi.importActual<typeof import("../../lib/api")>("../../lib/api");
    return { ...actual, apiFetch: (...args: unknown[]) => apiFetchMock(...args) };
});

const currencies: CurrencyMetadata[] = [
    { code: "CAD", displayName: "Canadian Dollar", minorUnitDigits: 2, amountDigits: 2 },
    { code: "TWD", displayName: "New Taiwan Dollar", minorUnitDigits: 2, amountDigits: 2 },
];

const initialSettings: GroupCurrencySettings = {
    settlementPreviewCurrency: "CAD",
    currencies: [
        {
            currency: "CAD",
            enabledForNewExpenses: true,
            historical: false,
            hasCurrentBalance: false,
            previewRate: "1",
        },
        {
            currency: "TWD",
            enabledForNewExpenses: true,
            historical: false,
            hasCurrentBalance: false,
            previewRate: "0.04",
        },
    ],
};

function Harness({
    settings = initialSettings,
    availableCurrencies = currencies,
}: {
    settings?: GroupCurrencySettings;
    availableCurrencies?: CurrencyMetadata[];
}) {
    const [value, setValue] = useState(settings);
    return (
        <>
            <CurrencySettingsEditor
                currencies={availableCurrencies}
                settings={value}
                onChange={setValue}
            />
            <output data-testid="settings">{JSON.stringify(value)}</output>
        </>
    );
}

function currentSettings() {
    return JSON.parse(screen.getByTestId("settings").textContent ?? "") as GroupCurrencySettings;
}

describe("CurrencySettingsEditor", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        apiFetchMock.mockImplementation((_path: string, init?: RequestInit) => {
            const payload = JSON.parse(String(init?.body)) as {
                sourceCurrencies: string[];
                settlementPreviewCurrency: string;
            };
            const cadRates: Record<string, string> = { CAD: "1", TWD: "0.043", USD: "1.35" };
            const targetRate = cadRates[payload.settlementPreviewCurrency];
            const recommendations = payload.sourceCurrencies.map((sourceCurrency) => ({
                sourceCurrency,
                settlementPreviewCurrency: payload.settlementPreviewCurrency,
                recommendedRate: sourceCurrency === payload.settlementPreviewCurrency
                    ? "1"
                    : payload.settlementPreviewCurrency === "CAD"
                        ? cadRates[sourceCurrency]
                        : sourceCurrency === "CAD"
                            ? "23.255813953488372"
                            : String(Number(cadRates[sourceCurrency]) / Number(targetRate)),
                rateSnapshotAt: "2026-09-22",
            }));
            return Promise.resolve(new Response(JSON.stringify({
                providerName: "Frankfurter",
                attributionUrl: "https://frankfurter.dev",
                recommendations,
            }), { status: 200, headers: { "Content-Type": "application/json" } }));
        });
    });

    afterEach(cleanup);

    it("disables without removing and removes only through the trash action", () => {
        render(<Harness />);

        fireEvent.click(screen.getByRole("checkbox", { name: /TWD/ }));
        expect(currentSettings().currencies).toHaveLength(2);
        expect(currentSettings().currencies[1].enabledForNewExpenses).toBe(false);

        fireEvent.click(screen.getByRole("button", { name: "Remove TWD" }));
        expect(currentSettings().currencies.map(({ currency }) => currency)).toEqual(["CAD"]);
    });

    it("keeps retained currencies and does not offer a remove action", () => {
        const retainedSettings: GroupCurrencySettings = {
            ...initialSettings,
            currencies: initialSettings.currencies.map((row) =>
                row.currency === "TWD" ? { ...row, historical: true } : row
            ),
        };
        render(<Harness settings={retainedSettings} />);

        expect(screen.queryByRole("button", { name: "Remove TWD" })).not.toBeInTheDocument();
        expect(screen.getByText(/Previously used or currently owed/)).toBeVisible();
    });

    it("shows an actionable recommendation card and stores its canonical rate", async () => {
        render(<Harness />);

        fireEvent.click(screen.getByRole("button", { name: "CAD → TWD" }));
        expect(await screen.findByText("1 CAD = 23.255814 TWD")).toBeVisible();
        fireEvent.click(screen.getByRole("button", { name: "Apply recommended rate for TWD" }));

        expect(screen.getByLabelText("1 CAD = … TWD")).toHaveValue("23.255814");
        expect(currentSettings().currencies[1].previewRate).toBe("0.043");
    });

    it("converts reverse manual input to the canonical stored direction", () => {
        render(<Harness />);

        fireEvent.click(screen.getByRole("button", { name: "CAD → TWD" }));
        fireEvent.change(screen.getByLabelText("1 CAD = … TWD"), {
            target: { value: "23.255813953488372" },
        });

        expect(currentSettings().currencies[1].previewRate).toBe("0.043");
    });

    it("switches preview currency using cached recommendations", async () => {
        render(<Harness settings={{
            ...initialSettings,
            currencies: initialSettings.currencies.map((row) =>
                row.currency === "TWD" ? { ...row, enabledForNewExpenses: false } : row
            ),
        }} />);

        await screen.findByText("1 TWD = 0.043 CAD");
        expect(apiFetchMock).toHaveBeenCalledTimes(1);
        fireEvent.click(screen.getByRole("button", { name: "Currency" }));
        fireEvent.click(screen.getByRole("option", { name: /TWD — New Taiwan Dollar/ }));

        await waitFor(() => {
            const settings = currentSettings();
            expect(settings.settlementPreviewCurrency).toBe("TWD");
            expect(settings.currencies).toEqual([
                expect.objectContaining({ currency: "CAD", previewRate: null }),
                expect.objectContaining({ currency: "TWD", enabledForNewExpenses: true, previewRate: "1" }),
            ]);
        });
        expect(await screen.findByText("1 CAD = 23.255814 TWD")).toBeVisible();
        expect(apiFetchMock).toHaveBeenCalledTimes(1);
    });

    it("requests only a newly selected currency and merges it into the cache", async () => {
        const supportedCurrencies: CurrencyMetadata[] = [
            ...currencies,
            { code: "USD", displayName: "US Dollar", minorUnitDigits: 2, amountDigits: 2 },
        ];
        render(<Harness
            availableCurrencies={supportedCurrencies}
            settings={{ settlementPreviewCurrency: "CAD", currencies: [initialSettings.currencies[0]] }}
        />);

        await waitFor(() => expect(screen.getByRole("button", { name: "Add currency" })).toBeEnabled());
        fireEvent.click(screen.getByRole("button", { name: "Add currency" }));
        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledTimes(1));
        expect(JSON.parse(String(apiFetchMock.mock.calls[0][1].body)).sourceCurrencies).toEqual(["CAD", "TWD"]);

        await waitFor(() => expect(screen.getByRole("button", { name: "Add currency" })).toBeEnabled());
        fireEvent.click(screen.getByRole("button", { name: "Add currency" }));
        await waitFor(() => expect(apiFetchMock).toHaveBeenCalledTimes(2));
        expect(JSON.parse(String(apiFetchMock.mock.calls[1][1].body)).sourceCurrencies).toEqual(["CAD", "USD"]);
    });
});
