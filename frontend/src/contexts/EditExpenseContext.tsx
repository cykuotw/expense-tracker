import {
    useState,
    useEffect,
    ReactNode,
    FormEvent,
    ReactElement,
    ChangeEvent,
} from "react";
import { useParams } from "react-router-dom";
import { API_URL } from "../configs/config";
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

const emptyData: expenseFormData = {
    groupId: "",
    expenseType: "",
    description: "",
    total: 0,
    currency: "",
    splitRule: Rule.Equally,
    payerUserId: "",
    ledgers: [],
};

export const EditExpenseProvider = ({ children }: { children: ReactNode }) => {
    const { id: expenseId = "" } = useParams();

    // handle form submission
    const [feedback, setFeedback] = useState<string>("");
    const [indicatorShow, setIndicatorShow] = useState<boolean>(false);

    const [formData, setFormData] = useState<expenseFormData>(emptyData);

    const handleUpdateExpense = async (e: FormEvent) => {
        e.preventDefault();

        try {
            setIndicatorShow(true);
            setFeedback("");

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

            const response = await fetch(`${API_URL}/expense/${expenseId}`, {
                method: "PUT",
                credentials: "include",
                body: JSON.stringify(payload),
            });
            if (!response.ok) {
                const errorMessage = await response.text();
                throw new Error(`Failed to update expense: ${errorMessage}`);
            }

            window.location.href = `/expense/${expenseId}`;
        } catch (error) {
            console.error("Error updating expense:", error);
            setFeedback("Error updating expense");
        } finally {
            setIndicatorShow(false);
        }
    };

    // handle page load
    const [groupList, setGroupList] = useState<GroupListItem[]>([]);
    const [expTypeOptions, setExpTypeOptions] = useState<ReactElement[]>([]);
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);

    useEffect(() => {
        const fetchGroupList = async () => {
            const response = await fetch(`${API_URL}/groups`, {
                method: "GET",
                credentials: "include",
            });
            if (!response.ok) return;

            const data: GroupListItem[] = await response.json();

            setGroupList(data);
            setFormData((prev) => ({
                ...prev,
                groupId: data[0].id,
            }));
        };

        const fetchExpeseTypes = async () => {
            const response = await fetch(`${API_URL}/expense_types`, {
                method: "GET",
                credentials: "include",
            });
            if (!response.ok) return;

            const data: ExpenseTypeItem[] = await response.json();
            // create options
            const options: ReactElement[] = [];
            let lastCategory = "";
            data.forEach((type) => {
                if (lastCategory !== type.category) {
                    options.push(
                        <option
                            disabled
                            key={type.category}
                            className="text-lg font-black font-mono"
                        >
                            {type.category}
                        </option>
                    );
                    lastCategory = type.category;
                }

                options.push(
                    <option value={type.id} key={type.id}>
                        {type.name}
                    </option>
                );
            });
            setExpTypeOptions(options);
        };

        const fetchExpenseDetail = async () => {
            const response = await fetch(`${API_URL}/expense/${expenseId}`, {
                method: "GET",
                credentials: "include",
            });
            if (!response.ok) return;

            const data: ExpenseDetailData = await response.json();

            setFormData((prev) => ({
                ...prev,
                groupId: data.groupId,
                expenseType: data.expenseTypeId,
                description: data.description,
                total: parseFloat(data.total),
                currency: data.currency,
                splitRule: data.splitRule as Rule,
                payerUserId: data.ledgers[0].lenderUserId,
                ledgers: data.ledgers.map((ledger) => ({
                    id: ledger.id,
                    userId: ledger.borrowerUserId,
                    share: parseFloat(ledger.share),
                })),
            }));

            const responseGroupMember = await fetch(
                `${API_URL}/group_member/${data.groupId}`,
                {
                    method: "GET",
                    credentials: "include",
                }
            );
            if (!responseGroupMember.ok) return;

            const groupMember: GroupMember[] = await responseGroupMember.json();

            setGroupMembers(groupMember);
        };

        fetchGroupList();
        fetchExpeseTypes();
        fetchExpenseDetail();
    }, [expenseId]);

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

        if (formData.splitRule !== Rule.Unequally) {
            setDataOk(totalOk && descriptionOk);
            return;
        }

        const ledgerTotal = formData.ledgers.reduce(
            (acc, ledger) => acc + ledger.share,
            0
        );
        const ledgerOk =
            ledgerTotal === formData.total &&
            formData.ledgers.every((ledger) => ledger.share >= 0);

        setDataOk(totalOk && descriptionOk && ledgerOk);
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
                expTypeOptions,
                groupMembers,
                feedback,
                indicatorShow,
                dataOk,
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
