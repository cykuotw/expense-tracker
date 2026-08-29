import { useState, useEffect, useRef, ReactNode, FormEvent, ReactElement } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "react-hot-toast";
import { apiFetch, asArray, getResponseErrorMessage } from "../lib/api";
import { ExpenseCreateData, ExpenseTypeItem } from "../types/expense";
import { GroupListItem, GroupMember } from "../types/group";
import { LedgerCreateData } from "../types/ledger";
import { Rule } from "../types/splitRule";
import { CreateExpenseContext } from "../hooks/CreateExpenseContextHooks";

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
    const total = Number(totalInput);
    const [description, setDescription] = useState<string>("");
    const [currency, setCurrency] = useState<string>("CAD");
    const [payer, setPayer] = useState<string>("");
    const [selectedRule, setSelectedRule] = useState<Rule>(Rule.Equally);
    const [ledgers, setLedgers] = useState<{ userId: string; share: number }[]>(
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
            const currencyPrecision: Record<"CAD" | "USD" | "NTD", number> = {
                CAD: 2,
                USD: 2,
                NTD: 0,
            };
            const precision: number =
                currencyPrecision[currency as keyof typeof currencyPrecision];
            const submissionLedgers = ledgers.map((ledger) => ({ ...ledger }));
            switch (selectedRule) {
                case Rule.Equally:
                case Rule.YouHalf:
                case Rule.OtherHalf: {
                    const peopleCount: number = submissionLedgers.length;

                    const split: number =
                        Math.floor((total / peopleCount) * 10 ** precision) /
                        10 ** precision;
                    const remaining: number = total - split * (peopleCount - 1);

                    const remainderIndex = 0;
                    for (let i = 0; i < peopleCount; i++) {
                        submissionLedgers[i].share = i === remainderIndex ? remaining : split;
                    }
                    break;
                }

                case Rule.YouFull:
                    submissionLedgers[0].share = 0;
                    submissionLedgers[1].share = total;
                    break;

                case Rule.OtherFull:
                    submissionLedgers[0].share = total;
                    submissionLedgers[1].share = 0;
                    break;

                default:
                    break;
            }

            const payload: ExpenseCreateData = {
                description: description,
                groupId: selectedGroupId || "",
                payByUserId: payer,
                expTypeId: selectedExpenseTypeId,
                total: total.toFixed(precision),
                currency: currency,
                splitRule: selectedRule,
                ledgers: submissionLedgers.map(
                    (ledger) =>
                        ({
                            borrowerUserId: ledger.userId,
                            lenderUserId: payer,
                            share: ledger.share.toFixed(precision),
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
    const [expTypeOptions, setExpTypeOptions] = useState<ReactElement[]>([]);
    const [groupMembers, setGroupMembers] = useState<GroupMember[]>([]);

    useEffect(() => {
        const fetchGroupList = async () => {
            const response = await apiFetch("/groups", {
                method: "GET",
            });
            if (!response.ok) return;

            const data = asArray<GroupListItem>(await response.json());
            setGroupList(data);
        };
        const fetchExpeseTypes = async () => {
            const response = await apiFetch("/expense_types", {
                method: "GET",
            });
            if (!response.ok) return;

            const data = asArray<ExpenseTypeItem>(await response.json());
            // create options
            const options: ReactElement[] = [];
            let lastCategory = "";
            let generalId = "";
            data.forEach((type) => {
                if (type.name === "General") {
                    generalId = type.id;
                }
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

            if (generalId !== "") {
                setSelectedExpenseTypeId(generalId);
            }
        };
        const fetchGroupMembers = async () => {
            const response = await apiFetch(`/group_member/${groupId}`, {
                method: "GET",
            });
            if (!response.ok) return;

            const data = asArray<GroupMember>(await response.json());

            setGroupMembers(data);
            setPayer(data[data.length - 1]?.userId ?? "");
            setLedgers(
                data.map((member) => ({
                    userId: member.userId,
                    share: 0,
                }))
            );
            if (data.length === 2) {
                setSelectedRule(Rule.YouHalf);
            } else {
                setSelectedRule(Rule.Equally);
            }
        };

        fetchGroupList();
        fetchExpeseTypes();
        fetchGroupMembers();
    }, [groupId]);

    // check input validity
    useEffect(() => {
        const totalOk = Number.isFinite(total) && total > 0;
        const descriptionOk = description.length > 0;

        if (selectedRule !== Rule.Unequally) {
            setDataOk(totalOk && descriptionOk);
            return;
        }

        const ledgerTotal = ledgers.reduce(
            (acc, ledger) => acc + ledger.share,
            0
        );
        const ledgerOk =
            ledgerTotal === total &&
            ledgers.every((ledger) => ledger.share >= 0);

        setDataOk(totalOk && descriptionOk && ledgerOk);
        setLedgerShareOk(ledgerOk);
        setLedgerShareMessage(
            ledgerOk
                ? `Total $0 ${currency} left.`
                : `Total $${(total - ledgerTotal).toFixed(2)} ${currency} left.`
        );
    }, [total, description, ledgers, selectedRule, currency]);

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
                currency,
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
                expTypeOptions,
                groupMembers,
                handleCreateExpense,
            }}
        >
            {children}
        </CreateExpenseContext.Provider>
    );
};
