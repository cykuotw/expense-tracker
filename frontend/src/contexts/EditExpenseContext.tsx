import {
    useState,
    useEffect,
    ReactNode,
    FormEvent,
    ChangeEvent,
} from "react";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { apiFetch, asArray, getResponseErrorMessage } from "../lib/api";
import {
    ExpenseDetailData,
    ExpenseTypeItem,
    ExpenseUpdateData,
} from "../types/expense";
import { GroupListItem, GroupMember } from "../types/group";
import { LedgerUpdateData } from "../types/ledger";
import { Rule } from "../types/splitRule";
import {
    EditExpenseContext,
    expenseFormData,
} from "../hooks/EditExpenseContextHooks";
import { isDateOnly } from "../lib/dateOnly";

const emptyData: expenseFormData = {
    groupId: "",
    expenseType: "",
    description: "",
    occurredOn: "",
    total: 0,
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
    const [initialFormData, setInitialFormData] =
        useState<expenseFormData | null>(null);

    const handleUpdateExpense = async (e: FormEvent) => {
        e.preventDefault();
        if (!dataOk || !hasChanges) return;

        try {
            setIndicatorShow(true);

            // set up ledgers in defult split rules
            const currencyPrecision: Record<"CAD" | "USD" | "NTD", number> = {
                CAD: 2,
                USD: 2,
                NTD: 0,
            };
            const precision: number =
                currencyPrecision[
                    formData.currency as keyof typeof currencyPrecision
                ];

            switch (formData.splitRule) {
                case Rule.Equally:
                case Rule.YouHalf:
                case Rule.OtherHalf: {
                    const peopleCount: number = formData.ledgers.length;

                    const split: number =
                        Math.floor(
                            (formData.total / peopleCount) * 10 ** precision
                        ) /
                        10 ** precision;
                    const remaining: number =
                        formData.total - split * (peopleCount - 1);

                    const randIndex = Math.floor(Math.random() * peopleCount);
                    for (let i = 0; i < peopleCount; i++) {
                        formData.ledgers[i].share =
                            i === randIndex ? remaining : split;
                    }
                    break;
                }

                case Rule.YouFull:
                    formData.ledgers[0].share = 0;
                    formData.ledgers[1].share = formData.total;
                    break;

                case Rule.OtherFull:
                    formData.ledgers[0].share = formData.total;
                    formData.ledgers[1].share = 0;
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
                total: formData.total.toFixed(precision),
                currency: formData.currency,
                splitRule: formData.splitRule,
                ledgers: formData.ledgers.map(
                    (ledger) =>
                        ({
                            ledgerId: ledger.id,
                            borrowerUserId: ledger.userId,
                            lenderUserId: formData.payerUserId,
                            share: ledger.share.toFixed(precision),
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

    useEffect(() => {
        setInitialFormData(null);

        const fetchGroupList = async () => {
            const response = await apiFetch("/groups", {
                method: "GET",
            });
            if (!response.ok) return;

            const data = asArray<GroupListItem>(await response.json());

            setGroupList(data);
            setFormData((prev) => ({
                ...prev,
                groupId: data[0]?.id ?? prev.groupId,
            }));
        };

        const fetchExpeseTypes = async () => {
            const response = await apiFetch("/expense_types", {
                method: "GET",
            });
            if (!response.ok) return;

            const data = asArray<ExpenseTypeItem>(await response.json());
            setExpenseTypes(data);
        };

        const fetchExpenseDetail = async () => {
            try {
                const response = await apiFetch(`/expense/${expenseId}`, {
                    method: "GET",
                });
                if (!response.ok) return;

                const responseData: unknown = await response.json();
                if (
                    typeof responseData !== "object" ||
                    responseData === null ||
                    Array.isArray(responseData)
                ) {
                    toast.error(LOAD_EXPENSE_FALLBACK);
                    return;
                }

                const expenseDetail = responseData as ExpenseDetailData;
                const data = {
                    ...expenseDetail,
                    items: asArray<ExpenseDetailData["items"][number]>(
                        expenseDetail.items
                    ),
                    ledgers: asArray<ExpenseDetailData["ledgers"][number]>(
                        expenseDetail.ledgers
                    ),
                };

                const nextFormData: expenseFormData = {
                    groupId: data.groupId,
                    expenseType: data.expenseTypeId,
                    description: data.description,
                    occurredOn: data.occurredOn,
                    total: parseFloat(data.total),
                    currency: data.currency,
                    splitRule: data.splitRule as Rule,
                    payerUserId: data.ledgers[0]?.lenderUserId ?? "",
                    ledgers: data.ledgers.map((ledger) => ({
                        id: ledger.id,
                        userId: ledger.borrowerUserId,
                        share: parseFloat(ledger.share),
                    })),
                };
                setFormData(nextFormData);
                setInitialFormData(cloneExpenseFormData(nextFormData));

                const responseGroupMember = await apiFetch(
                    `/group_member/${data.groupId}`,
                    {
                        method: "GET",
                    }
                );
                if (!responseGroupMember.ok) return;

                const groupMember = asArray<GroupMember>(
                    await responseGroupMember.json()
                );

                setGroupMembers(groupMember);
            } catch {
                toast.error(LOAD_EXPENSE_FALLBACK);
            }
        };

        fetchGroupList();
        fetchExpeseTypes();
        fetchExpenseDetail();
    }, [expenseId]);

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
        const totalOk = formData.total > 0;
        const descriptionOk = formData.description.length > 0;
        const occurredOnOk = isDateOnly(formData.occurredOn);

        if (formData.splitRule !== Rule.Unequally) {
            setDataOk(totalOk && descriptionOk && occurredOnOk);
            return;
        }

        const ledgerTotal = formData.ledgers.reduce(
            (acc, ledger) => acc + ledger.share,
            0
        );
        const ledgerOk =
            ledgerTotal === formData.total &&
            formData.ledgers.every((ledger) => ledger.share >= 0);

        setDataOk(totalOk && descriptionOk && occurredOnOk && ledgerOk);
        setLedgerShareOk(ledgerOk);
        setLedgerShareMessage(
            ledgerOk
                ? `Total $0 ${formData.currency} left.`
                : `Total $${(formData.total - ledgerTotal).toFixed(2)} ${
                      formData.currency
                  } left.`
        );
    }, [formData]);

    return (
        <EditExpenseContext.Provider
            value={{
                formData,
                setFormData,
                groupList,
                expenseTypes,
                groupMembers,
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
