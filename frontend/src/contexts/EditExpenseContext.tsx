import {
    ChangeEvent,
    FormEvent,
    ReactNode,
    useCallback,
    useEffect,
    useMemo,
    useState,
} from "react";
import { useNavigate, useParams } from "react-router-dom";
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
const UPDATE_EXPENSE_FALLBACK = "Error updating expense";
const LOAD_EXPENSE_FALLBACK = "Failed to load expense.";

function cloneExpenseFormData(formData: expenseFormData): expenseFormData {
    return {
        ...formData,
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
    const navigate = useNavigate();
    const { id: expenseId = "" } = useParams();
    const [indicatorShow, setIndicatorShow] = useState(false);
    const [mainFormVisited, setMainFormVisited] = useState(false);
    const [formData, setFormData] =
        useState<expenseFormData>(EMPTY_FORM_DATA);
    const [currencies, setCurrencies] = useState<CurrencyMetadata[]>([]);
    const [initialFormData, setInitialFormData] =
        useState<expenseFormData | null>(null);
    const [groupList, setGroupList] = useState<GroupListItem[]>([]);
    const [expenseTypes, setExpenseTypes] = useState<ExpenseTypeItem[]>([]);
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);
    const [currentUserId, setCurrentUserId] = useState("");
    const [groupMembersLoadStatus, setGroupMembersLoadStatus] =
        useState<GroupMembersLoadStatus>("idle");
    const [optionsReloadVersion, setOptionsReloadVersion] = useState(0);

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
    const dataOk =
        totalUnits !== null &&
        totalUnits > 0n &&
        formData.description.length > 0 &&
        isDateOnly(formData.occurredOn) &&
        allocationCalculation.valid &&
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
            !dataOk ||
            !hasChanges ||
            amountDigits === null ||
            totalUnits === null
        ) {
            return;
        }

        try {
            setIndicatorShow(true);
            const payload: ExpenseUpdateData = {
                description: formData.description,
                occurredOn: formData.occurredOn,
                groupId: formData.groupId,
                payByUserId: formData.payerUserId,
                expTypeId: formData.expenseType,
                total: unitsToDecimal(totalUnits, amountDigits),
                currency: formData.currency,
                allocation: formData.allocation,
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
                    expenseType: data.expenseTypeId,
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

                setGroupList(asArray<GroupListItem>(options.groups));
                setExpenseTypes(
                    asArray<ExpenseTypeItem>(options.expenseTypes)
                );
                setCurrencies(currencyOptions);
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
                amountDigits,
                setFormData,
                groupList,
                expenseTypes,
                groupMembers,
                currentUserId,
                groupMembersLoadStatus,
                reloadGroupMembers,
                indicatorShow,
                dataOk,
                hasChanges,
                allocationCalculation,
                mainFormVisited,
                markMainFormVisited,
                handleUpdateExpense,
                handleFormDataChange,
            }}
        >
            {children}
        </EditExpenseContext.Provider>
    );
};
