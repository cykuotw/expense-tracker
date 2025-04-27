import { ReactElement, useEffect, useState } from "react";
import { useSearchParams, Link } from "react-router-dom";

import Icon from "@mdi/react";
import {
    mdiCamera,
    mdiCheckBold,
    mdiCheckCircleOutline,
    mdiSubdirectoryArrowLeft,
} from "@mdi/js";

import { API_URL } from "../configs/config";
import { ExpenseCreateData, ExpenseTypeItem } from "../types/expense";
import { GroupListItem, GroupMember } from "../types/group";
import { LedgerCreateData } from "../types/ledger";
import { Rule } from "../types/splitRule";

const CreateExpense = () => {
    const [searchParams] = useSearchParams();
    const groupId = searchParams.get("g");

    // handle form submission
    const [feedback, setFeedback] = useState<string>("");
    const [indicatorShow, setIndicatorShow] = useState<boolean>(false);

    const [selectedGroupId, setSelectedGroupId] = useState<string | null>(
        groupId
    );
    const [selectedExpenseTypeId, setSelectedExpenseTypeId] =
        useState<string>("");
    const [total, setTotal] = useState<number>(0);
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

    const handleCreateExpense = async (e: React.FormEvent) => {
        e.preventDefault();

        setIndicatorShow(true);
        setFeedback("");

        // set up ledgers in defult split rules
        const currencyPrecision: Record<"CAD" | "USD" | "NTD", number> = {
            CAD: 2,
            USD: 2,
            NTD: 0,
        };
        const precision: number =
            currencyPrecision[currency as keyof typeof currencyPrecision];
        switch (selectedRule) {
            case Rule.Equally:
            case Rule.YouHalf:
            case Rule.OtherHalf: {
                const peopleCount: number = ledgers.length;

                const split: number =
                    Math.floor((total / peopleCount) * 10 ** precision) /
                    10 ** precision;
                const remaining: number = total - split * (peopleCount - 1);

                const randIndex = Math.floor(Math.random() * peopleCount);
                for (let i = 0; i < peopleCount; i++) {
                    ledgers[i].share = i === randIndex ? remaining : split;
                }
                break;
            }

            case Rule.YouFull:
                ledgers[0].share = 0;
                ledgers[1].share = total;
                break;

            case Rule.OtherFull:
                ledgers[0].share = total;
                ledgers[1].share = 0;
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
            ledgers: ledgers.map(
                (ledger) =>
                    ({
                        borrowerUserId: ledger.userId,
                        lenderUserId: payer,
                        share: ledger.share.toFixed(precision),
                    } as LedgerCreateData)
            ),
        };

        const response = await fetch(`${API_URL}/create_expense`, {
            method: "POST",
            credentials: "include",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(payload),
        });

        setIndicatorShow(false);

        if (!response.ok) {
            setFeedback("Failed to create expense.");
            return;
        }
        setFeedback("Your expense has been created!");
    };

    // handle on page load
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
            const response = await fetch(`${API_URL}/group_member/${groupId}`, {
                method: "GET",
                credentials: "include",
            });
            if (!response.ok) return;

            const data: GroupMember[] = await response.json();

            setGroupMembers(data);
            setPayer(data[data.length - 1].userId);
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
    }, []);

    // check input validity
    useEffect(() => {
        const totalOk = total > 0;
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
    }, [total, description, ledgers, selectedRule]);

    return (
        <div className="flex flex-col justify-center items-center py-5 w-screen">
            <form
                className="flex flex-col justify-center items-center py-5 space-y-5 md:w-1/3 w-5/6m max-w-md"
                onSubmit={handleCreateExpense}
            >
                <div className="text-2xl">Add Expense</div>

                {/* GROUP SELECTOR */}
                <select
                    className="select select-bordered w-full text-base text-center"
                    id="groupId"
                    name="groupId"
                    value={selectedGroupId || ""}
                    onChange={(e) => setSelectedGroupId(e.target.value)}
                >
                    {groupList.map((group) => (
                        <option key={group.id} value={group.id}>
                            {group.groupName}
                        </option>
                    ))}
                </select>

                {/* EXPENSE TYPE SELECTOR */}
                <select
                    className="select select-bordered w-full text-base text-center"
                    id="expenseType"
                    name="expenseType"
                    value={selectedExpenseTypeId}
                    onChange={(e) => {
                        console.log(e.target.value);
                        setSelectedExpenseTypeId(e.target.value);
                    }}
                >
                    {expTypeOptions}
                </select>

                {/* DESCRIPTION INPUT */}
                <label className="input input-bordered flex items-center w-full">
                    <input
                        type="text"
                        name="description"
                        className="grow"
                        placeholder="Description"
                        value={description}
                        onChange={(e) => {
                            setDescription(e.target.value);
                        }}
                        required
                    />
                </label>

                <div className="flex flex-row justify-start items-start w-full">
                    {/* CURRENCY SELECTOR */}
                    <select
                        className="select select-bordered w-1/3 text-base text-center"
                        name="currency"
                        value={currency}
                        onChange={(e) => {
                            setCurrency(e.target.value);
                        }}
                    >
                        <option>CAD</option>
                        <option>NTD</option>
                        <option>USD</option>
                    </select>

                    {/* AMOUNT INPUT */}
                    <label className="input input-bordered flex items-center w-full">
                        <input
                            type="number"
                            name="total"
                            className="grow"
                            step="0.001"
                            placeholder="0.00"
                            value={total}
                            onChange={(e) => {
                                setTotal(parseFloat(e.target.value) || 0.0);
                            }}
                            required
                            min={0}
                        />
                    </label>
                </div>

                {/* RECEIPT UPLOAD BUTTON */}
                <div className="hidden">
                    <label
                        style={{ display: "inline-block" }}
                        className="w-2/3 h-12 border border-gray-400 rounded-full bg-base-100 hover:bg-base-300"
                    >
                        <input type="file" style={{ display: "none" }} />
                        <div className="flex flex-row items-center justify-center h-full space-x-3">
                            <Icon path={mdiCamera} size={1} />
                            <p>Upload Receipt</p>
                        </div>
                    </label>
                </div>

                {/* SPLIT RULE SELECTOR */}
                {groupMembers.length <= 1 ? (
                    <></>
                ) : groupMembers.length === 2 ? (
                    // 2 members
                    <>
                        <select
                            className="select select-bordered w-full text-base text-center"
                            name="splitRule"
                            value={selectedRule}
                            onChange={(e) => {
                                setSelectedRule(e.target.value as Rule);
                                switch (e.target.value) {
                                    case Rule.YouHalf:
                                    case Rule.YouFull:
                                        setPayer(
                                            groupMembers[
                                                groupMembers.length - 1
                                            ].userId
                                        );
                                        break;
                                    case Rule.OtherHalf:
                                    case Rule.OtherFull:
                                        setPayer(groupMembers[0].userId);
                                        break;
                                    default:
                                        break;
                                }
                            }}
                        >
                            <option value={Rule.YouHalf}>
                                You paid, split equally
                            </option>
                            <option value={Rule.YouFull}>
                                You are owed the full amount
                            </option>
                            <option value={Rule.OtherHalf}>
                                {groupMembers[0].username} paid, split euqally
                            </option>
                            <option value={Rule.OtherFull}>
                                {groupMembers[0].username} is owed the full
                                amount
                            </option>
                            <option value={Rule.Unequally}>Unequally</option>
                        </select>
                        <div className="hidden flex-row justify-center items-center w-full space-x-2">
                            <p className="w-max">Paid by</p>
                            <select
                                className="select select-sm select-bordered w-max border-dashed"
                                name="payer"
                            >
                                <option
                                    value={
                                        groupMembers[groupMembers.length - 1]
                                            ?.userId
                                    }
                                >
                                    You
                                </option>
                                <option value={groupMembers[0].userId}>
                                    {groupMembers[0].userId}
                                </option>
                            </select>
                        </div>
                    </>
                ) : (
                    // N members
                    <div className="flex flex-row justify-center items-center w-full space-x-2">
                        <p className="w-max">Paid by</p>
                        <select
                            className="select select-sm select-bordered w-max border-dashed"
                            name="payer"
                            value={payer}
                            onChange={(e) => {
                                setPayer(e.target.value);
                            }}
                        >
                            <option
                                value={
                                    groupMembers[groupMembers.length - 1].userId
                                }
                                key={
                                    groupMembers[groupMembers.length - 1].userId
                                }
                            >
                                You
                            </option>
                            {groupMembers.map((member) => {
                                if (
                                    member.userId !==
                                    groupMembers[groupMembers.length - 1].userId
                                ) {
                                    return (
                                        <option
                                            value={member.userId}
                                            key={member.userId}
                                        >
                                            {member.username}
                                        </option>
                                    );
                                }
                            })}
                        </select>
                        <p className="w-max">and split</p>
                        <select
                            className="select select-sm select-bordered w-max border-dashed"
                            name="splitRule"
                            value={selectedRule}
                            onChange={(e) => {
                                setSelectedRule(e.target.value as Rule);
                            }}
                        >
                            <option value={Rule.Equally}>Equally</option>
                            <option value={Rule.Unequally}>Unequally</option>
                        </select>
                    </div>
                )}

                {/* LEDGERS - FOR UNEQUAL SPLIT RULE */}
                <div
                    className={`${
                        selectedRule === Rule.Unequally ? "" : "hidden"
                    } flex-col justify-center items-center w-full space-y-1`}
                >
                    {ledgers.map((ledger, index) => (
                        <div
                            className="flex items-center w-full"
                            key={ledger.userId}
                        >
                            <p className="w-1/3 text-right mr-2">
                                {
                                    groupMembers.find(
                                        (member) =>
                                            member.userId === ledger.userId
                                    )?.username
                                }
                                :
                            </p>

                            <label className="input input-bordered flex items-center w-2/3 gap-2">
                                Share:
                                <input
                                    type="number"
                                    className="grow"
                                    step="0.001"
                                    placeholder="0.00"
                                    value={ledger.share}
                                    onChange={(e) => {
                                        const updated = [...ledgers];
                                        updated[index].share =
                                            parseFloat(e.target.value) || 0;
                                        setLedgers(updated);
                                    }}
                                />
                            </label>
                        </div>
                    ))}
                    <div className="flex flex-col items-center w-full">
                        <p
                            className={`${
                                ledgerShareOk
                                    ? "text-green-700"
                                    : "text-red-700"
                            }`}
                        >
                            {ledgerShareMessage}
                        </p>
                    </div>
                </div>

                {/* SUBMIT BUTTON */}
                <button
                    type="submit"
                    className="btn btn-active btn-neutral btn-wide text-lg font-light"
                    {...(dataOk ? {} : { disabled: true })}
                >
                    <Icon path={mdiCheckBold} size={1} />
                    OK
                </button>

                {/* FEEDBACK */}
                <div className={`${indicatorShow ? "" : "hidden"}`}>
                    <div className="flex justify-center items-center w-full">
                        <span className="loading loading-spinner loading-md"></span>
                    </div>
                </div>
                <div className={`${feedback.length === 0 ? "hidden" : ""}`}>
                    <div className="animate-fade">
                        <div role="alert" className="alert alert-success">
                            <Icon path={mdiCheckCircleOutline} size={1} />
                            <span>{feedback}</span>
                        </div>
                    </div>
                </div>
            </form>
            <div className="flex justify-center w-full mt-4">
                <Link className="btn btn-ghost" to={`/group/${groupId}`}>
                    <Icon path={mdiSubdirectoryArrowLeft} size={1} />
                    Back to Group
                </Link>
            </div>
        </div>
    );
};

export default CreateExpense;
