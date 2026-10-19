import {
    ChangeEvent,
    FormEvent,
    ReactNode,
    useCallback,
    useEffect,
    useMemo,
    useRef,
    useState,
} from "react";
import { useParams } from "react-router-dom";
import { useReturnNavigation } from "../hooks/navigation";
import { toast } from "react-hot-toast";
import {
    EditExpenseContext,
    expenseFormData,
} from "../hooks/EditExpenseContextHooks";
import {
    apiFetch,
    asArray,
    getResponseErrorMessage,
} from "../lib/api";
import {
    getExpenseSubmissionErrorMessage,
    getExpenseSubmissionFallback,
} from "../lib/expenseSubmissionError";
import { calculateExpenseAllocation } from "../lib/expenseAllocation";
import { isDateOnly } from "../lib/dateOnly";
import {
    CurrencyMetadata,
    currencyAmountDigits,
    decimalToUnits,
    unitsToDecimal,
} from "../lib/money";
import {
    EditExpenseOptionsData,
    ExpenseDetailData,
    ExpenseTypeItem,
    ExpenseUpdateData,
} from "../types/expense";
import {
    GroupListItem,
    GroupMember,
    GroupMembersLoadStatus,
} from "../types/group";
import { legacyCurrencySettings } from "../lib/currencySettings";
import ArchivedGroupMessage from "../components/group/ArchivedGroupMessage";

import { validateReceiptExpense } from "../lib/receiptDraft";
import { calculateCategoryBreakdown, categoryBreakdownPayload } from "../lib/categoryBreakdown";

const EMPTY_FORM_DATA: expenseFormData = {
    groupId: "",
    expenseType: "",
    description: "",
    occurredOn: "",
    total: "",
    currency: "",
    allocation: { mode: "equal", participants: [] },
    payerUserId: "",
};
const LOAD_EXPENSE_FALLBACK = "Failed to load expense.";

function cloneExpenseFormData(formData: expenseFormData): expenseFormData {
    return {
        ...formData,
        categoryBreakdown: formData.categoryBreakdown?.map(row => ({ ...row })),
        allocation: {
            ...formData.allocation,
            participants: formData.allocation.participants.map(
                (participant) => ({ ...participant })
            ),
        },
    };
}

function isSameExpenseFormData(
    left: expenseFormData,
    right: expenseFormData
): boolean {
    return JSON.stringify(left) === JSON.stringify(right);
}

export const EditExpenseProvider = ({ children }: { children: ReactNode }) => {
    const { finish } = useReturnNavigation("/");
    const { id: expenseId = "" } = useParams();
    const [indicatorShow, setIndicatorShow] = useState(false);
    const [submissionError, setSubmissionError] = useState<string | null>(null);
    const submissionInFlightRef = useRef(false);
    const originalItemIDs = useRef(new Set<string>());
    const originalExpense = useRef<ExpenseDetailData | null>(null);
    const [mainFormVisited, setMainFormVisited] = useState(false);
    const [formData, setFormData] =
        useState<expenseFormData>(EMPTY_FORM_DATA);
    const [receipt, setReceipt] = useState<ExpenseDetailData["receipt"]>(undefined);
    const [currencies, setCurrencies] = useState<CurrencyMetadata[]>([]);
    const [enabledCurrencies, setEnabledCurrencies] = useState<CurrencyMetadata[]>([]);
    const [initialFormData, setInitialFormData] =
        useState<expenseFormData | null>(null);
    const [groupList, setGroupList] = useState<GroupListItem[]>([]);
    const [expenseTypes, setExpenseTypes] = useState<ExpenseTypeItem[]>([]);
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);
    const [currentUserId, setCurrentUserId] = useState("");
    const [groupMembersLoadStatus, setGroupMembersLoadStatus] =
        useState<GroupMembersLoadStatus>("idle");
    const [optionsReloadVersion, setOptionsReloadVersion] = useState(0);
    const [groupArchived, setGroupArchived] = useState(false);

    const amountDigits = currencyAmountDigits(currencies, formData.currency);
    const allocationCalculation = useMemo(
        () =>
            calculateExpenseAllocation(
                formData.total,
                amountDigits,
                formData.allocation,
                formData.currency
            ),
        [
            amountDigits,
            formData.allocation,
            formData.currency,
            formData.total,
        ]
    );
    const totalUnits =
        amountDigits === null
            ? null
            : decimalToUnits(formData.total, amountDigits);
    const receiptValidation = formData.receiptDetails ? validateReceiptExpense({ ...formData.receiptDetails, total: formData.total }, amountDigits) : null;
    const dataOk =
        !groupArchived &&
        (receiptValidation?.valid ?? true) &&
        totalUnits !== null &&
        totalUnits > 0n &&
        formData.description.length > 0 &&
        isDateOnly(formData.occurredOn) &&
        allocationCalculation.valid &&
        calculateCategoryBreakdown(formData.categoryBreakdown ?? [], formData.total, amountDigits).valid &&
        groupMembersLoadStatus === "ready" &&
        Boolean(formData.payerUserId && formData.expenseType);
    const hasChanges =
        initialFormData !== null &&
        !isSameExpenseFormData(formData, initialFormData);

    const reloadGroupMembers = useCallback(() => {
        setOptionsReloadVersion((version) => version + 1);
    }, []);
    const markMainFormVisited = useCallback(() => {
        setMainFormVisited(true);
    }, []);

    const handleUpdateExpense = async (event: FormEvent) => {
        event.preventDefault();
        if (
            submissionInFlightRef.current ||
            !dataOk ||
            !hasChanges ||
            amountDigits === null ||
            totalUnits === null
        ) {
            return;
        }

        submissionInFlightRef.current = true;
        setSubmissionError(null);
        setIndicatorShow(true);
        try {
            const payload: ExpenseUpdateData = {
                description: formData.description,
                occurredOn: formData.occurredOn,
                groupId: formData.groupId,
                payByUserId: formData.payerUserId,
                expTypeId: formData.categoryBreakdown?.[0]?.expenseTypeId ?? formData.expenseType,
                categoryBreakdown: categoryBreakdownPayload(formData.categoryBreakdown ?? [], amountDigits),
                total: unitsToDecimal(totalUnits, amountDigits),
                currency: formData.currency,
                allocation: formData.allocation,
                providerName: formData.receiptDetails?.merchant ?? formData.providerName,
                invoiceUrl: formData.invoiceUrl,
                ...(!formData.receiptDetails && originalExpense.current?.currency === formData.currency && decimalToUnits(originalExpense.current.total, amountDigits) === totalUnits ? {
                    subTotal: originalExpense.current.subTotal, taxFeeTip: originalExpense.current.taxFeeTip,
                } : {}),
                ...(receiptValidation?.valid && formData.receiptDetails ? {
                    subTotal: receiptValidation.subTotal,
                    taxFeeTip: receiptValidation.taxFeeTip,
                    items: receiptValidation.items?.map((item, index) => ({ ...item, ...(originalItemIDs.current.has(formData.receiptDetails!.items[index].id) ? { itemId: formData.receiptDetails!.items[index].id } : {}) })),
                } : {}),
            };
            const response = await apiFetch(`/expense/${expenseId}`, {
                method: "PUT",
                body: JSON.stringify(payload),
            });
            if (!response.ok) {
                setSubmissionError(
                    await getExpenseSubmissionErrorMessage(response, "update")
                );
                return;
            }

            toast.success("Expense updated", { duration: 1000 });
            window.setTimeout(() => {
                finish(`/expense/${expenseId}`);
            }, 1000);
        } catch {
            setSubmissionError(getExpenseSubmissionFallback("update"));
        } finally {
            submissionInFlightRef.current = false;
            setIndicatorShow(false);
        }
    };

    useEffect(() => {
        const abortController = new AbortController();
        let active = true;
        setInitialFormData(null);
        setGroupMembers([]);
        setCurrentUserId("");
        setGroupMembersLoadStatus("loading");

        const fetchExpenseDetail = async () => {
            try {
                const response = await apiFetch(
                    `/expense/${expenseId}/edit-options`,
                    { method: "GET", signal: abortController.signal }
                );
                if (!response.ok) {
                    const message = await getResponseErrorMessage(
                        response,
                        LOAD_EXPENSE_FALLBACK
                    );
                    if (active) toast.error(message);
                    return;
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
                    responseData as Partial<EditExpenseOptionsData>;
                if (!options.expense || !options.group) {
                    throw new Error("missing expense options");
                }
                const expenseDetail = options.expense;
                const group = options.group;
                if (active) setGroupArchived(group.isActive === false);
                const currencyOptions = asArray<CurrencyMetadata>(
                    options.currencies
                );
                const members = asArray<GroupMember>(group.members);
                if (
                    members.length === 0 ||
                    !group.currentUserId ||
                    !members.some(
                        ({ userId }) => userId === group.currentUserId
                    ) ||
                    !expenseDetail.currency ||
                    currencyAmountDigits(
                        currencyOptions,
                        expenseDetail.currency
                    ) === null
                ) {
                    throw new Error("invalid expense options");
                }
                const data = {
                    ...expenseDetail,
                    items: asArray<ExpenseDetailData["items"][number]>(
                        expenseDetail.items
                    ),
                    ledgers: asArray<ExpenseDetailData["ledgers"][number]>(
                        expenseDetail.ledgers
                    ),
                };
                if (
                    !data.allocation ||
                    !Array.isArray(data.allocation.participants)
                ) {
                    throw new Error("missing expense allocation");
                }
                const nextFormData: expenseFormData = {
                    groupId: data.groupId,
                    providerName: data.providerName,
                    invoiceUrl: data.invoiceUrl,
                    ...(data.items.length > 0 || (decimalToUnits(data.taxFeeTip || "0", currencyAmountDigits(currencyOptions, data.currency)!) ?? 0n) > 0n ? {
                        receiptDetails: {
                            merchant: data.providerName ?? "", subtotal: data.subTotal, tax: data.taxFeeTip, tip: "",
                            items: data.items.map(item => ({ id: item.itemId, description: item.description ?? item.itemName,
                                quantity: item.quantity ?? "", unit: item.unit ?? "", unitPrice: item.unitPrice ?? "", lineTotal: item.lineTotal ?? item.itemSubTotal })),
                        },
                    } : {}),
                    expenseType: data.expenseTypeId,
                    categoryBreakdown: (data.categoryBreakdown?.length ?? 0) > 1 ? data.categoryBreakdown!.map(row => ({ ...row, clientId: crypto.randomUUID() })) : [],
                    description: data.description,
                    occurredOn: data.occurredOn,
                    total: data.total,
                    currency: data.currency,
                    allocation: {
                        ...data.allocation,
                        participants: data.allocation.participants.map(
                            (participant) => ({ ...participant })
                        ),
                    },
                    payerUserId: data.ledgers[0]?.lenderUserId ?? "",
                };
                if (!active) return;

                originalExpense.current = data;
                originalItemIDs.current = new Set(data.items.map(item => item.itemId));
                setReceipt(data.receipt);
                setGroupList(asArray<GroupListItem>(options.groups));
                setExpenseTypes(
                    asArray<ExpenseTypeItem>(options.expenseTypes)
                );
                setCurrencies(currencyOptions);
                const groupCurrencySettings =
                    group.currencySettings ?? legacyCurrencySettings(group.currency);
                const enabledCodes = new Set(
                    groupCurrencySettings.currencies
                        .filter(({ enabledForNewExpenses }) => enabledForNewExpenses)
                        .map(({ currency }) => currency)
                );
                enabledCodes.add(expenseDetail.currency);
                setEnabledCurrencies(
                    currencyOptions.filter(({ code }) => enabledCodes.has(code))
                );
                setGroupMembers(members);
                setCurrentUserId(group.currentUserId);
                setGroupMembersLoadStatus("ready");
                setFormData(nextFormData);
                setInitialFormData(cloneExpenseFormData(nextFormData));
            } catch {
                if (active) {
                    setGroupMembersLoadStatus("error");
                    toast.error(LOAD_EXPENSE_FALLBACK);
                }
            }
        };

        void fetchExpenseDetail();
        return () => {
            active = false;
            abortController.abort();
        };
    }, [expenseId, optionsReloadVersion]);

    const handleFormDataChange = (
        event: ChangeEvent<HTMLSelectElement | HTMLInputElement>
    ) => {
        const { name, value } = event.target;
        setFormData((current) => ({ ...current, [name]: value }));
    };

    return (
        <EditExpenseContext.Provider
            value={{
                formData,
                receipt,
                amountDigits,
                currencies: enabledCurrencies,
                setFormData,
                groupList,
                expenseTypes,
                groupMembers,
                currentUserId,
                groupMembersLoadStatus,
                reloadGroupMembers,
                indicatorShow,
                submissionError,
                dataOk,
                hasChanges,
                allocationCalculation,
                mainFormVisited,
                markMainFormVisited,
                handleUpdateExpense,
                handleFormDataChange,
            }}
        >
            {groupArchived ? <ArchivedGroupMessage groupId={formData.groupId} /> : children}
        </EditExpenseContext.Provider>
    );
};
