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
import {
    apiFetch,
    asArray,
    getResponseErrorMessage,
} from "../lib/api";
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

const CREATE_EXPENSE_FALLBACK = "Failed to create expense.";
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
    const [searchParams] = useSearchParams();
    const groupId = searchParams.get("g");
    const [indicatorShow, setIndicatorShow] = useState(false);
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
    const [payer, setPayer] = useState("");
    const [allocation, setAllocation] =
        useState<ExpenseAllocation>(EMPTY_ALLOCATION);

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
    const dataOk =
        totalUnits !== null &&
        totalUnits > 0n &&
        description.length > 0 &&
        isDateOnly(occurredOn) &&
        allocationCalculation.valid &&
        groupMembersLoadStatus === "ready" &&
        Boolean(selectedGroupId && payer && selectedExpenseTypeId);

    const reloadGroupMembers = useCallback(() => {
        setGroupMembersReloadVersion((version) => version + 1);
    }, []);
    const markMainFormVisited = useCallback(() => {
        setMainFormVisited(true);
    }, []);

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
                toast.error(
                    await getResponseErrorMessage(
                        response,
                        CREATE_EXPENSE_FALLBACK
                    )
                );
                return;
            }

            submissionIntentRef.current = null;
            toast.success("Your expense has been created!", { duration: 1000 });
            if (selectedGroupId) {
                navigate(`/group/${selectedGroupId}`);
            }
        } catch {
            toast.error(CREATE_EXPENSE_FALLBACK);
        } finally {
            submissionInFlightRef.current = false;
            setIndicatorShow(false);
        }
    };

    useEffect(() => {
        setGroupMembersLoadStatus(selectedGroupId ? "loading" : "idle");
        if (!selectedGroupId) {
            setGroupMembers([]);
            setCurrentUserId("");
            setPayer("");
            setAllocation(EMPTY_ALLOCATION);
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
                    (!group?.currency ||
                        currencyAmountDigits(
                            currencyOptions,
                            group.currency
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
                if (!selectedGroupId || !group) return;

                setGroupMembers(members);
                setCurrentUserId(group.currentUserId);
                setGroupMembersLoadStatus("ready");
                setCurrency(group.currency);
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
                amountDigits,
                payer,
                setPayer,
                allocation,
                setAllocation,
                allocationCalculation,
                mainFormVisited,
                markMainFormVisited,
                indicatorShow,
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
