import {
    FormEvent,
    ReactNode,
    useCallback,
    useEffect,
    useMemo,
    useRef,
    useState,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { CreateExpenseContext } from "../hooks/CreateExpenseContextHooks";
import { useAuth } from "../hooks/AuthContextHooks";
import {
    apiFetch,
    asArray,
} from "../lib/api";
import {
    getExpenseSubmissionErrorMessage,
    getExpenseSubmissionFallback,
} from "../lib/expenseSubmissionError";
import { calculateExpenseAllocation } from "../lib/expenseAllocation";
import { isDateOnly, todayDateOnly } from "../lib/dateOnly";
import {
    CurrencyMetadata,
    currencyAmountDigits,
    decimalToUnits,
    unitsToDecimal,
} from "../lib/money";
import { ExpenseAllocation } from "../types/allocation";
import {
    CreateExpenseOptionsData,
    ExpenseCreateData,
    ExpenseTypeItem,
} from "../types/expense";
import {
    GroupListItem,
    GroupMember,
    GroupMembersLoadStatus,
} from "../types/group";
import { legacyCurrencySettings } from "../lib/currencySettings";
import { requestReceiptDraft, ReceiptOCRError } from "../lib/receiptOcr";
import { validateReceiptExpense } from "../lib/receiptDraft";
import type { PreparedReceipt } from "../lib/receiptEditor";
import type { AccountSettingsData } from "../types/account";
import type { ConfirmedExpenseItem, OCRDraft, ReviewedReceiptDraft } from "../types/ocr";

const EMPTY_ALLOCATION: ExpenseAllocation = {
    mode: "equal",
    participants: [],
};

export const CreateExpenseProvider = ({
    children,
}: {
    children: ReactNode;
}) => {
    const navigate = useNavigate();
    const { userID } = useAuth();
    const [searchParams] = useSearchParams();
    const groupId = searchParams.get("g");
    const [indicatorShow, setIndicatorShow] = useState(false);
    const [submissionError, setSubmissionError] = useState<string | null>(null);
    const submissionInFlightRef = useRef(false);
    const submissionIntentRef = useRef<{
        key: string;
        fingerprint: string;
    } | null>(null);
    const [mainFormVisited, setMainFormVisited] = useState(false);

    const [selectedGroupId, setSelectedGroupId] = useState<string | null>(
        groupId
    );
    const [selectedExpenseTypeId, setSelectedExpenseTypeId] = useState("");
    const [totalInput, setTotalInput] = useState("");
    const [description, setDescription] = useState("");
    const [occurredOn, setOccurredOn] = useState(todayDateOnly);
    const [currency, setCurrency] = useState("CAD");
    const [currencies, setCurrencies] = useState<CurrencyMetadata[]>([]);
    const [enabledCurrencies, setEnabledCurrencies] = useState<CurrencyMetadata[]>([]);
    const [payer, setPayer] = useState("");
    const [allocation, setAllocation] =
        useState<ExpenseAllocation>(EMPTY_ALLOCATION);
    const [merchant, setMerchant] = useState("");
    const [subtotalInput, setSubtotalInput] = useState("");
    const [taxInput, setTaxInput] = useState("");
    const [tipInput, setTipInput] = useState("");
    const [items, setItems] = useState<ConfirmedExpenseItem[]>([]);
    const [receiptOCREnabled, setReceiptOCREnabled] = useState<boolean | null>(null);
    const [ocrStatus, setOCRStatus] = useState<"idle" | "scanning" | "ready" | "error">("idle");
    const [ocrDraft, setOCRDraft] = useState<OCRDraft | null>(null);
    const [ocrError, setOCRError] = useState<string | null>(null);
    const ocrAbortRef = useRef<AbortController | null>(null);
    const previousGroupRef = useRef(selectedGroupId);

    const [groupList, setGroupList] = useState<GroupListItem[]>([]);
    const [expenseTypes, setExpenseTypes] = useState<ExpenseTypeItem[]>([]);
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);
    const [currentUserId, setCurrentUserId] = useState("");
    const [groupMembersLoadStatus, setGroupMembersLoadStatus] =
        useState<GroupMembersLoadStatus>("idle");
    const [groupMembersReloadVersion, setGroupMembersReloadVersion] =
        useState(0);

    const amountDigits = currencyAmountDigits(currencies, currency);
    const allocationCalculation = useMemo(
        () =>
            calculateExpenseAllocation(
                totalInput,
                amountDigits,
                allocation,
                currency
            ),
        [allocation, amountDigits, currency, totalInput]
    );
    const totalUnits =
        amountDigits === null
            ? null
            : decimalToUnits(totalInput, amountDigits);
    const receiptDetailsActive = Boolean(
        merchant.trim() || subtotalInput.trim() || taxInput.trim() ||
        tipInput.trim() || items.length > 0
    );
    const receiptValidation = validateReceiptExpense({
        merchant, subtotal: subtotalInput, tax: taxInput, tip: tipInput,
        total: totalInput, items,
    }, amountDigits);
    const dataOk =
        totalUnits !== null &&
        totalUnits > 0n &&
        description.length > 0 &&
        isDateOnly(occurredOn) &&
        allocationCalculation.valid &&
        receiptValidation.valid &&
        groupMembersLoadStatus === "ready" &&
        Boolean(selectedGroupId && payer && selectedExpenseTypeId);

    const reloadGroupMembers = useCallback(() => {
        setGroupMembersReloadVersion((version) => version + 1);
    }, []);
    const markMainFormVisited = useCallback(() => {
        setMainFormVisited(true);
    }, []);

    const clearReceiptWorkflow = useCallback(() => {
        ocrAbortRef.current?.abort("cancelled");
        ocrAbortRef.current = null;
        setOCRStatus("idle");
        setOCRDraft(null);
        setOCRError(null);
    }, []);

    const startReceiptOCR = useCallback(async (receipt: PreparedReceipt) => {
        if (!userID || receiptOCREnabled !== true) {
            setOCRStatus("error");
            setOCRError("Receipt scanning is not available for this account.");
            return;
        }
        ocrAbortRef.current?.abort("replaced");
        const controller = new AbortController();
        ocrAbortRef.current = controller;
        setOCRStatus("scanning");
        setOCRDraft(null);
        setOCRError(null);
        try {
            const draft = await requestReceiptDraft(receipt, userID, controller.signal);
            if (ocrAbortRef.current !== controller) return;
            setOCRDraft(draft);
            setOCRStatus("ready");
        } catch (error) {
            if (ocrAbortRef.current !== controller) return;
            const receiptError = error instanceof ReceiptOCRError ? error : null;
            if (receiptError?.code === "receipt_ocr_not_granted") {
                setReceiptOCREnabled(false);
            }
            setOCRError(receiptError?.message ?? "The receipt could not be scanned. Continue with manual entry.");
            setOCRStatus("error");
        } finally {
            if (ocrAbortRef.current === controller) ocrAbortRef.current = null;
        }
    }, [receiptOCREnabled, userID]);

    const applyReviewedReceipt = useCallback((draft: ReviewedReceiptDraft) => {
        setMerchant(draft.merchant);
        setDescription((current) => current.trim() ? current : draft.merchant);
        setOccurredOn(draft.date);
        setSubtotalInput(draft.subtotal);
        setTaxInput(draft.tax);
        setTipInput(draft.tip);
        setTotalInput(draft.total);
        setItems(draft.items);
        clearReceiptWorkflow();
    }, [clearReceiptWorkflow]);

    const handleCreateExpense = async (event: FormEvent) => {
        event.preventDefault();
        if (
            submissionInFlightRef.current ||
            !dataOk ||
            amountDigits === null ||
            totalUnits === null
        ) {
            return;
        }

        submissionInFlightRef.current = true;
        setSubmissionError(null);
        setIndicatorShow(true);
        try {
            const normalizedTotal = unitsToDecimal(totalUnits, amountDigits);
            const payload: ExpenseCreateData = {
                description,
                groupId: selectedGroupId ?? "",
                payByUserId: payer,
                expTypeId: selectedExpenseTypeId,
                total: normalizedTotal,
                currency,
                allocation,
                occurredOn,
                providerName: merchant.trim(),
                ...(receiptDetailsActive ? {
                    subTotal: receiptValidation.subTotal,
                    taxFeeTip: receiptValidation.taxFeeTip,
                    items: receiptValidation.items,
                } : {}),
            };

            const fingerprint = JSON.stringify(payload);
            if (submissionIntentRef.current?.fingerprint !== fingerprint) {
                submissionIntentRef.current = {
                    key: crypto.randomUUID(),
                    fingerprint,
                };
            }
            const response = await apiFetch("/create_expense", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "Idempotency-Key": submissionIntentRef.current.key,
                },
                body: fingerprint,
            });
            if (!response.ok) {
                setSubmissionError(
                    await getExpenseSubmissionErrorMessage(response, "create")
                );
                return;
            }

            submissionIntentRef.current = null;
            clearReceiptWorkflow();
            toast.success("Your expense has been created!", { duration: 1000 });
            if (selectedGroupId) {
                navigate(`/group/${selectedGroupId}`);
            }
        } catch {
            setSubmissionError(getExpenseSubmissionFallback("create"));
        } finally {
            submissionInFlightRef.current = false;
            setIndicatorShow(false);
        }
    };

    useEffect(() => {
        const controller = new AbortController();
        const loadCapability = async () => {
            try {
                const response = await apiFetch("/account", { signal: controller.signal });
                if (!response.ok) throw new Error("account capability request failed");
                const account = (await response.json()) as Partial<AccountSettingsData>;
                setReceiptOCREnabled(account.capabilities?.receiptOcr === true);
            } catch {
                if (!controller.signal.aborted) setReceiptOCREnabled(false);
            }
        };
        void loadCapability();
        return () => controller.abort();
    }, []);

    useEffect(() => () => {
        ocrAbortRef.current?.abort("unmounted");
    }, []);

    useEffect(() => {
        if (previousGroupRef.current === selectedGroupId) return;
        previousGroupRef.current = selectedGroupId;
        clearReceiptWorkflow();
        setMerchant("");
        setSubtotalInput("");
        setTaxInput("");
        setTipInput("");
        setItems([]);
    }, [clearReceiptWorkflow, selectedGroupId]);

    useEffect(() => {
        setGroupMembersLoadStatus(selectedGroupId ? "loading" : "idle");
        if (!selectedGroupId) {
            setGroupMembers([]);
            setCurrentUserId("");
            setPayer("");
            setAllocation(EMPTY_ALLOCATION);
            setEnabledCurrencies([]);
        }

        const abortController = new AbortController();
        let active = true;
        const fetchOptions = async () => {
            try {
                const query = selectedGroupId
                    ? `?groupId=${encodeURIComponent(selectedGroupId)}`
                    : "";
                const response = await apiFetch(
                    `/expense_create_options${query}`,
                    { method: "GET", signal: abortController.signal }
                );
                if (!response.ok) {
                    throw new Error("expense options request failed");
                }
                const responseData: unknown = await response.json();
                if (
                    typeof responseData !== "object" ||
                    responseData === null ||
                    Array.isArray(responseData)
                ) {
                    throw new Error("invalid expense options response");
                }

                const options =
                    responseData as Partial<CreateExpenseOptionsData>;
                const groups = asArray<GroupListItem>(options.groups);
                const types = asArray<ExpenseTypeItem>(options.expenseTypes);
                const currencyOptions = asArray<CurrencyMetadata>(
                    options.currencies
                );
                const group = options.group;
                const groupCurrencySettings = group
                    ? group.currencySettings ?? legacyCurrencySettings(group.currency)
                    : null;
                const members = asArray<GroupMember>(group?.members);
                if (
                    selectedGroupId &&
                    (!group ||
                        members.length === 0 ||
                        !group.currentUserId ||
                        !members.some(
                            ({ userId }) => userId === group.currentUserId
                        ))
                ) {
                    throw new Error("group detail has no members");
                }
                if (
                    selectedGroupId &&
                    (!groupCurrencySettings?.settlementPreviewCurrency ||
                        currencyAmountDigits(
                            currencyOptions,
                            groupCurrencySettings.settlementPreviewCurrency
                        ) === null)
                ) {
                    throw new Error("group currency metadata is unavailable");
                }
                if (!active) return;

                setGroupList(groups);
                setExpenseTypes(types);
                setCurrencies(currencyOptions);
                const generalId = types.find(
                    ({ name }) => name === "General"
                )?.id;
                if (generalId) {
                    setSelectedExpenseTypeId(
                        (current) => current || generalId
                    );
                }
                if (!selectedGroupId || !group || !groupCurrencySettings) return;

                setGroupMembers(members);
                setCurrentUserId(group.currentUserId);
                setGroupMembersLoadStatus("ready");
                const enabledCodes = new Set(
                    groupCurrencySettings.currencies
                        .filter(({ enabledForNewExpenses }) => enabledForNewExpenses)
                        .map(({ currency: code }) => code)
                );
                const nextCurrencies = currencyOptions.filter(({ code }) => enabledCodes.has(code));
                setEnabledCurrencies(nextCurrencies);
                setCurrency(
                    enabledCodes.has(groupCurrencySettings.settlementPreviewCurrency)
                        ? groupCurrencySettings.settlementPreviewCurrency
                        : nextCurrencies[0]?.code ?? ""
                );
                setPayer(group.currentUserId);
                setAllocation({
                    mode: "equal",
                    participants: members.map(({ userId }) => ({ userId })),
                });
            } catch {
                if (active && selectedGroupId) {
                    setGroupMembersLoadStatus("error");
                }
            }
        };

        void fetchOptions();
        return () => {
            active = false;
            abortController.abort();
        };
    }, [groupMembersReloadVersion, selectedGroupId]);

    return (
        <CreateExpenseContext.Provider
            value={{
                groupId,
                selectedGroupId,
                setSelectedGroupId,
                selectedExpenseTypeId,
                setSelectedExpenseTypeId,
                total: totalInput,
                totalInput,
                setTotalInput,
                description,
                setDescription,
                occurredOn,
                setOccurredOn,
                currency,
                currencies: enabledCurrencies,
                setCurrency,
                amountDigits,
                payer,
                setPayer,
                allocation,
                setAllocation,
                allocationCalculation,
                mainFormVisited,
                markMainFormVisited,
                merchant,
                setMerchant,
                subtotalInput,
                setSubtotalInput,
                taxInput,
                setTaxInput,
                tipInput,
                setTipInput,
                items,
                setItems,
                receiptDetailsActive,
                receiptOCREnabled,
                ocrStatus,
                ocrDraft,
                ocrError,
                startReceiptOCR,
                applyReviewedReceipt,
                clearReceiptWorkflow,
                indicatorShow,
                submissionError,
                dataOk,
                groupList,
                expenseTypes,
                groupMembers,
                currentUserId,
                groupMembersLoadStatus,
                reloadGroupMembers,
                handleCreateExpense,
            }}
        >
            {children}
        </CreateExpenseContext.Provider>
    );
};
