import {
    useCallback,
    useState,
    useEffect,
    useRef,
    ReactNode,
    FormEvent,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import {
    apiFetch,
    asArray,
    getResponseErrorMessage,
} from "../lib/api";
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
import { LedgerCreateData } from "../types/ledger";
import { Rule } from "../types/splitRule";
import { CreateExpenseContext } from "../hooks/CreateExpenseContextHooks";
import { isDateOnly, todayDateOnly } from "../lib/dateOnly";
import {
    CurrencyMetadata,
    currencyAmountDigits,
    decimalToUnits,
    splitEqually,
    unitsToDecimal,
} from "../lib/money";

const CREATE_EXPENSE_FALLBACK = "Failed to create expense.";

export const CreateExpenseProvider = ({
    children,
}: {
    children: ReactNode;
}) => {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const groupId = searchParams.get("g");

    // handle form submission
    const [indicatorShow, setIndicatorShow] = useState<boolean>(false);
    const submissionInFlightRef = useRef(false);
    const submissionIntentRef = useRef<{ key: string; fingerprint: string } | null>(null);

    const [selectedGroupId, setSelectedGroupId] = useState<string | null>(
        groupId
    );
    const [selectedExpenseTypeId, setSelectedExpenseTypeId] =
        useState<string>("");
    const [totalInput, setTotalInput] = useState("");
    const total = totalInput;
    const [description, setDescription] = useState<string>("");
    const [occurredOn, setOccurredOn] = useState(todayDateOnly);
    const [currency, setCurrency] = useState<string>("CAD");
    const [currencies, setCurrencies] = useState<CurrencyMetadata[]>([]);
    const amountDigits = currencyAmountDigits(currencies, currency);
    const [payer, setPayer] = useState<string>("");
    const [selectedRule, setSelectedRule] = useState<Rule>(Rule.Equally);
    const [ledgers, setLedgers] = useState<{ userId: string; share: string }[]>(
        []
    );

    const [ledgerShareOk, setLedgerShareOk] = useState<boolean>(false);
    const [ledgerShareMessage, setLedgerShareMessage] = useState<string>("");
    const [dataOk, setDataOk] = useState<boolean>(false);

    const handleCreateExpense = async (e: FormEvent) => {
        e.preventDefault();
        if (submissionInFlightRef.current) return;

        submissionInFlightRef.current = true;

        setIndicatorShow(true);

        try {
            // set up ledgers in defult split rules
            const precision = amountDigits;
            if (precision === null) {
                toast.error("Currency metadata is unavailable. Try loading the group again.");
                return;
            }
            const totalUnits = decimalToUnits(totalInput, precision);
            if (totalUnits === null || totalUnits <= 0n) {
                toast.error("Enter a valid amount.");
                return;
            }
            const submissionLedgers = ledgers.map((ledger) => ({ ...ledger }));
            switch (selectedRule) {
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

            const payload: ExpenseCreateData = {
                description: description,
                groupId: selectedGroupId || "",
                payByUserId: payer,
                expTypeId: selectedExpenseTypeId,
                total: unitsToDecimal(totalUnits, precision),
                currency: currency,
                splitRule: selectedRule,
                occurredOn,
                ledgers: submissionLedgers.map(
                    (ledger) =>
                        ({
                            borrowerUserId: ledger.userId,
                            lenderUserId: payer,
                            share: unitsToDecimal(decimalToUnits(ledger.share, precision) ?? 0n, precision),
                        } as LedgerCreateData)
                ),
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
                body: JSON.stringify(payload),
            });

            if (!response.ok) {
                toast.error(
                    await getResponseErrorMessage(
                        response,
                        CREATE_EXPENSE_FALLBACK
                    )
                );
                return;
            }
            submissionIntentRef.current = null;
            toast.success("Your expense has been created!", {
                duration: 1000,
            });
            const targetGroupId = selectedGroupId || groupId;
            if (targetGroupId) {
                navigate(`/group/${targetGroupId}`);
            }
        } catch {
            toast.error(CREATE_EXPENSE_FALLBACK);
        } finally {
            submissionInFlightRef.current = false;
            setIndicatorShow(false);
        }
    };

    // handle on page load
    const [groupList, setGroupList] = useState<GroupListItem[]>([]);
    const [expenseTypes, setExpenseTypes] = useState<ExpenseTypeItem[]>([]);
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);
    const [groupMembersLoadStatus, setGroupMembersLoadStatus] =
        useState<GroupMembersLoadStatus>("idle");
    const [groupMembersReloadVersion, setGroupMembersReloadVersion] =
        useState(0);
    const reloadGroupMembers = useCallback(() => {
        setGroupMembersReloadVersion((version) => version + 1);
    }, []);

    useEffect(() => {
        setGroupMembers([]);
        setGroupMembersLoadStatus(selectedGroupId ? "loading" : "idle");
        setPayer("");
        setLedgers([]);
        setSelectedRule(Rule.Equally);

        const abortController = new AbortController();
        let active = true;

        const fetchOptions = async () => {
            try {
                const query = selectedGroupId
                    ? `?groupId=${encodeURIComponent(selectedGroupId)}`
                    : "";
                const response = await apiFetch(
                    `/expense_create_options${query}`,
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
                    throw new Error("invalid expense options response");
                }

                const options = responseData as Partial<CreateExpenseOptionsData>;
                const groups = asArray<GroupListItem>(options.groups);
                const types = asArray<ExpenseTypeItem>(options.expenseTypes);
                const currencyOptions = asArray<CurrencyMetadata>(options.currencies);
                const group = options.group;
                const members = asArray<GroupMember>(group?.members);
                if (selectedGroupId && (!group || members.length === 0)) {
                    throw new Error("group detail has no members");
                }
                if (
                    selectedGroupId &&
                    (!group?.currency ||
                        currencyAmountDigits(currencyOptions, group.currency) === null)
                ) {
                    throw new Error("group currency metadata is unavailable");
                }
                if (!active) return;

                setGroupList(groups);
                setExpenseTypes(types);
                setCurrencies(currencyOptions);
                const generalId = types.find(
                    (type) => type.name === "General"
                )?.id;
                if (generalId) {
                    setSelectedExpenseTypeId((current) => current || generalId);
                }
                if (!selectedGroupId || !group) return;

                setGroupMembers(members);
                setGroupMembersLoadStatus("ready");
                if (typeof group.currency === "string" && group.currency) {
                    setCurrency(group.currency);
                }
                setPayer(members[members.length - 1]?.userId ?? "");
                setLedgers(
                    members.map((member) => ({
                        userId: member.userId,
                        share: "",
                    }))
                );
                if (members.length === 2) {
                    setSelectedRule(Rule.YouHalf);
                } else {
                    setSelectedRule(Rule.Equally);
                }
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

    // check input validity
    useEffect(() => {
        const precision = amountDigits;
        if (precision === null) {
            setDataOk(false);
            setLedgerShareOk(false);
            setLedgerShareMessage("Currency metadata is unavailable.");
            return;
        }
        const totalUnits = decimalToUnits(total, precision);
        const totalOk = totalUnits !== null && totalUnits > 0n;
        const descriptionOk = description.length > 0;
        const occurredOnOk = isDateOnly(occurredOn);

        if (selectedRule !== Rule.Unequally) {
            setDataOk(
                totalOk &&
                    descriptionOk &&
                    occurredOnOk &&
                    groupMembersLoadStatus === "ready" &&
                    Boolean(selectedGroupId && payer && selectedExpenseTypeId)
            );
            return;
        }

        const ledgerUnits = ledgers.map((ledger) => decimalToUnits(ledger.share || "0", precision));
        const ledgerTotal = ledgerUnits.reduce<bigint>((sum, value) => sum + (value ?? 0n), 0n);
        const ledgerOk =
            totalUnits !== null && ledgerTotal === totalUnits && ledgerUnits.every((value) => value !== null && value >= 0n);

        setDataOk(
            totalOk &&
                descriptionOk &&
                occurredOnOk &&
                ledgerOk &&
                groupMembersLoadStatus === "ready" &&
                Boolean(selectedGroupId && payer && selectedExpenseTypeId)
        );
        setLedgerShareOk(ledgerOk);
        setLedgerShareMessage(
            ledgerOk
                ? `Total 0 ${currency} left.`
                : `Total ${unitsToDecimal((totalUnits ?? 0n) - ledgerTotal, precision)} ${currency} left.`
        );
    }, [
        amountDigits,
        total,
        description,
        occurredOn,
        ledgers,
        selectedRule,
        currency,
        groupMembersLoadStatus,
        selectedGroupId,
        payer,
        selectedExpenseTypeId,
    ]);

    return (
        <CreateExpenseContext.Provider
            value={{
                groupId,
                selectedGroupId,
                setSelectedGroupId,
                selectedExpenseTypeId,
                setSelectedExpenseTypeId,
                total,
                totalInput,
                setTotalInput,
                description,
                setDescription,
                occurredOn,
                setOccurredOn,
                currency,
                amountDigits,
                setCurrency,
                payer,
                setPayer,
                selectedRule,
                setSelectedRule,
                ledgers,
                setLedgers,
                indicatorShow,
                dataOk,
                ledgerShareOk,
                ledgerShareMessage,
                groupList,
                expenseTypes,
                groupMembers,
                groupMembersLoadStatus,
                reloadGroupMembers,
                handleCreateExpense,
            }}
        >
            {children}
        </CreateExpenseContext.Provider>
    );
};
