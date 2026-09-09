import {
    useCallback,
    useState,
    useEffect,
    useRef,
    ReactNode,
    FormEvent,
    ChangeEvent,
} from "react";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import {
    apiFetch,
    asArray,
    getResponseErrorMessage,
} from "../lib/api";
import {
    ExpenseDetailData,
    EditExpenseOptionsData,
    ExpenseTypeItem,
    ExpenseUpdateData,
} from "../types/expense";
import {
    GroupListItem,
    GroupMember,
    GroupMembersLoadStatus,
} from "../types/group";
import { LedgerUpdateData } from "../types/ledger";
import { Rule } from "../types/splitRule";
import {
    EditExpenseContext,
    expenseFormData,
} from "../hooks/EditExpenseContextHooks";
import { isDateOnly } from "../lib/dateOnly";
import { orientTwoPersonRuleForViewer } from "../lib/expenseSplitRule";
import {
    CurrencyMetadata,
    currencyAmountDigits,
    decimalToUnits,
    splitEqually,
    unitsToDecimal,
} from "../lib/money";

const emptyData: expenseFormData = {
    groupId: "",
    expenseType: "",
    description: "",
    occurredOn: "",
    total: "",
    currency: "",
    splitRule: Rule.Equally,
    payerUserId: "",
    ledgers: [],
};

const UPDATE_EXPENSE_FALLBACK = "Error updating expense";
const LOAD_EXPENSE_FALLBACK = "Failed to load expense.";

function cloneExpenseFormData(formData: expenseFormData): expenseFormData {
    return {
        ...formData,
        ledgers: formData.ledgers.map((ledger) => ({ ...ledger })),
    };
}

function isSameExpenseFormData(
    left: expenseFormData,
    right: expenseFormData
): boolean {
    return (
        left.groupId === right.groupId &&
        left.expenseType === right.expenseType &&
        left.description === right.description &&
        left.occurredOn === right.occurredOn &&
        left.total === right.total &&
        left.currency === right.currency &&
        left.splitRule === right.splitRule &&
        left.payerUserId === right.payerUserId &&
        left.ledgers.length === right.ledgers.length &&
        left.ledgers.every((ledger, index) => {
            const compared = right.ledgers[index];
            return (
                compared !== undefined &&
                ledger.id === compared.id &&
                ledger.userId === compared.userId &&
                ledger.share === compared.share
            );
        })
    );
}

export const EditExpenseProvider = ({ children }: { children: ReactNode }) => {
    const navigate = useNavigate();
    const { id: expenseId = "" } = useParams();

    // handle form submission
    const [indicatorShow, setIndicatorShow] = useState<boolean>(false);

    const [formData, setFormData] = useState<expenseFormData>(emptyData);
    const [currencies, setCurrencies] = useState<CurrencyMetadata[]>([]);
    const amountDigits = currencyAmountDigits(currencies, formData.currency);
    const [initialFormData, setInitialFormData] =
        useState<expenseFormData | null>(null);

    const handleUpdateExpense = async (e: FormEvent) => {
        e.preventDefault();
        if (!dataOk || !hasChanges) return;

        try {
            setIndicatorShow(true);

            // set up ledgers in defult split rules
            const precision = amountDigits;
            if (precision === null) {
                toast.error("Currency metadata is unavailable. Try loading the expense again.");
                return;
            }
            const totalUnits = decimalToUnits(formData.total, precision);
            if (totalUnits === null || totalUnits <= 0n) return;
            const submissionLedgers = formData.ledgers.map((ledger) => ({ ...ledger }));

            switch (formData.splitRule) {
                case Rule.Equally:
                case Rule.YouHalf:
                case Rule.OtherHalf: {
                    const split = splitEqually(totalUnits, submissionLedgers.map((ledger) => ledger.userId));
                    if (!split) return;
                    submissionLedgers.forEach((ledger) => {
                        ledger.share = unitsToDecimal(split.get(ledger.userId)!, precision);
                    });
                    break;
                }

                case Rule.YouFull:
                    submissionLedgers[0].share = "0";
                    submissionLedgers[1].share = unitsToDecimal(totalUnits, precision);
                    break;

                case Rule.OtherFull:
                    submissionLedgers[0].share = unitsToDecimal(totalUnits, precision);
                    submissionLedgers[1].share = "0";
                    break;

                default:
                    break;
            }

            const payload: ExpenseUpdateData = {
                description: formData.description,
                occurredOn: formData.occurredOn,
                groupId: formData.groupId,
                payByUserId: formData.payerUserId,
                expTypeId: formData.expenseType,
                total: unitsToDecimal(totalUnits, precision),
                currency: formData.currency,
                splitRule: formData.splitRule,
                ledgers: submissionLedgers.map(
                    (ledger) =>
                        ({
                            ledgerId: ledger.id,
                            borrowerUserId: ledger.userId,
                            lenderUserId: formData.payerUserId,
                            share: unitsToDecimal(decimalToUnits(ledger.share, precision) ?? 0n, precision),
                        } as LedgerUpdateData)
                ),
            };

            const response = await apiFetch(`/expense/${expenseId}`, {
                method: "PUT",
                body: JSON.stringify(payload),
            });
            if (!response.ok) {
                toast.error(
                    await getResponseErrorMessage(
                        response,
                        UPDATE_EXPENSE_FALLBACK
                    )
                );
                return;
            }

            toast.success("Expense updated", { duration: 1000 });
            window.setTimeout(() => {
                navigate(`/expense/${expenseId}`);
            }, 1000);
        } catch (error) {
            console.error("Error updating expense:", error);
            toast.error(UPDATE_EXPENSE_FALLBACK);
        } finally {
            setIndicatorShow(false);
        }
    };

    // handle page load
    const [groupList, setGroupList] = useState<GroupListItem[]>([]);
    const [expenseTypes, setExpenseTypes] = useState<ExpenseTypeItem[]>([]);
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);
    const [groupMembersLoadStatus, setGroupMembersLoadStatus] =
        useState<GroupMembersLoadStatus>("idle");
    const initialLoadCompleteRef = useRef(false);
    const skipNextGroupLoadRef = useRef("");
    const [optionsReloadVersion, setOptionsReloadVersion] = useState(0);
    const [groupMembersReloadVersion, setGroupMembersReloadVersion] =
        useState(0);
    const reloadGroupMembers = useCallback(() => {
        if (initialLoadCompleteRef.current) {
            setGroupMembersReloadVersion((version) => version + 1);
        } else {
            setOptionsReloadVersion((version) => version + 1);
        }
    }, []);

    useEffect(() => {
        const abortController = new AbortController();
        let active = true;
        initialLoadCompleteRef.current = false;
        setInitialFormData(null);
        setGroupMembers([]);
        setGroupMembersLoadStatus("loading");

        const fetchExpenseDetail = async () => {
            try {
                const response = await apiFetch(
                    `/expense/${expenseId}/edit-options`,
                    {
                        method: "GET",
                        signal: abortController.signal,
                    }
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
                    if (active) toast.error(LOAD_EXPENSE_FALLBACK);
                    return;
                }

                const options = responseData as Partial<EditExpenseOptionsData>;
                if (!options.expense || !options.group) {
                    if (active) toast.error(LOAD_EXPENSE_FALLBACK);
                    return;
                }
                const expenseDetail = options.expense;
                const currencyOptions = asArray<CurrencyMetadata>(options.currencies);
                if (
                    !expenseDetail.currency ||
                    currencyAmountDigits(currencyOptions, expenseDetail.currency) === null
                ) {
                    throw new Error("expense currency metadata is unavailable");
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
                const members = asArray<GroupMember>(options.group.members);
                if (members.length === 0) {
                    throw new Error("expense options group has no members");
                }

                const payerUserId = data.ledgers[0]?.lenderUserId ?? "";
                const nextFormData: expenseFormData = {
                    groupId: data.groupId,
                    expenseType: data.expenseTypeId,
                    description: data.description,
                    occurredOn: data.occurredOn,
                    total: data.total,
                    currency: data.currency,
                    splitRule: orientTwoPersonRuleForViewer(
                        data.splitRule as Rule,
                        payerUserId,
                        data.currentUser
                    ),
                    payerUserId,
                    ledgers: data.ledgers.map((ledger) => ({
                        id: ledger.id,
                        userId: ledger.borrowerUserId,
                        share: ledger.share,
                    })),
                };
                if (active) {
                    skipNextGroupLoadRef.current = nextFormData.groupId;
                    initialLoadCompleteRef.current = true;
                    setGroupList(asArray<GroupListItem>(options.groups));
                    setExpenseTypes(asArray<ExpenseTypeItem>(options.expenseTypes));
                    setCurrencies(currencyOptions);
                    setGroupMembers(members);
                    setGroupMembersLoadStatus("ready");
                    setFormData(nextFormData);
                    setInitialFormData(cloneExpenseFormData(nextFormData));
                }

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

    useEffect(() => {
        if (!formData.groupId) {
            setGroupMembers([]);
            setGroupMembersLoadStatus("idle");
            return;
        }

        if (!initialLoadCompleteRef.current) return;

        if (skipNextGroupLoadRef.current === formData.groupId) {
            skipNextGroupLoadRef.current = "";
            return;
        }

        setGroupMembers([]);
        setGroupMembersLoadStatus("loading");

        const abortController = new AbortController();
        let active = true;

        const fetchGroupMembers = async () => {
            try {
                const response = await apiFetch(
                    `/expense/${expenseId}/edit-options?groupId=${encodeURIComponent(formData.groupId)}`,
                    {
                        method: "GET",
                        signal: abortController.signal,
                    }
                );
                if (!response.ok) throw new Error("expense options request failed");

                const responseData: unknown = await response.json();
                if (
                    typeof responseData !== "object" ||
                    responseData === null ||
                    Array.isArray(responseData)
                ) {
                    throw new Error("invalid group detail response");
                }

                const options = responseData as Partial<EditExpenseOptionsData>;
                const data = asArray<GroupMember>(options.group?.members);
                const currencyOptions = asArray<CurrencyMetadata>(options.currencies);
                if (data.length === 0) {
                    throw new Error("group detail has no members");
                }
                if (
                    !options.group?.currency ||
                    currencyAmountDigits(currencyOptions, options.group.currency) === null
                ) {
                    throw new Error("group currency metadata is unavailable");
                }
                if (!active) return;

                setGroupMembers(data);
                setCurrencies(currencyOptions);
                setGroupMembersLoadStatus("ready");
            } catch {
                if (active) setGroupMembersLoadStatus("error");
            }
        };

        void fetchGroupMembers();

        return () => {
            active = false;
            abortController.abort();
        };
    }, [expenseId, formData.groupId, groupMembersReloadVersion]);

    const hasChanges =
        initialFormData !== null &&
        !isSameExpenseFormData(formData, initialFormData);

    // handle form data update
    const handleFormDataChange = (
        e: ChangeEvent<HTMLSelectElement | HTMLInputElement>
    ) => {
        const { name, value } = e.target;
        setFormData((prev) => ({
            ...prev,
            [name]: value,
        }));
    };

    // handle form data validation
    const [ledgerShareOk, setLedgerShareOk] = useState<boolean>(false);
    const [ledgerShareMessage, setLedgerShareMessage] = useState<string>("");
    const [dataOk, setDataOk] = useState<boolean>(false);

    useEffect(() => {
        const precision = amountDigits;
        if (precision === null) {
            setDataOk(false);
            setLedgerShareOk(false);
            setLedgerShareMessage("Currency metadata is unavailable.");
            return;
        }
        const totalUnits = decimalToUnits(formData.total, precision);
        const totalOk = totalUnits !== null && totalUnits > 0n;
        const descriptionOk = formData.description.length > 0;
        const occurredOnOk = isDateOnly(formData.occurredOn);

        if (formData.splitRule !== Rule.Unequally) {
            setDataOk(
                totalOk &&
                    descriptionOk &&
                    occurredOnOk &&
                    groupMembersLoadStatus === "ready" &&
                    Boolean(formData.payerUserId && formData.expenseType)
            );
            return;
        }

        const ledgerUnits = formData.ledgers.map((ledger) => decimalToUnits(ledger.share || "0", precision));
        const ledgerTotal = ledgerUnits.reduce<bigint>((sum, value) => sum + (value ?? 0n), 0n);
        const ledgerOk =
            totalUnits !== null && ledgerTotal === totalUnits && ledgerUnits.every((value) => value !== null && value >= 0n);

        setDataOk(
            totalOk &&
                descriptionOk &&
                occurredOnOk &&
                ledgerOk &&
                groupMembersLoadStatus === "ready" &&
                Boolean(formData.payerUserId && formData.expenseType)
        );
        setLedgerShareOk(ledgerOk);
        setLedgerShareMessage(
            ledgerOk
                ? `Total 0 ${formData.currency} left.`
                : `Total ${unitsToDecimal((totalUnits ?? 0n) - ledgerTotal, precision)} ${formData.currency} left.`
        );
    }, [amountDigits, formData, groupMembersLoadStatus]);

    return (
        <EditExpenseContext.Provider
            value={{
                formData,
                amountDigits,
                setFormData,
                groupList,
                expenseTypes,
                groupMembers,
                groupMembersLoadStatus,
                reloadGroupMembers,
                indicatorShow,
                dataOk,
                hasChanges,
                ledgerShareOk,
                ledgerShareMessage,
                handleUpdateExpense,
                handleFormDataChange,
            }}
        >
            {children}
        </EditExpenseContext.Provider>
    );
};
