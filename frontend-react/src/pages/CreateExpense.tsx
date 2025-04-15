import { ReactElement, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { GroupListItem } from "../types/group";
import { API_URL } from "../configs/config";
import SplitRule from "../components/expense/SplitRule";
import Icon from "@mdi/react";
import { mdiCamera, mdiCheckBold, mdiCheckCircleOutline } from "@mdi/js";
import { ExpenseTypeItem } from "../types/expense";

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

    const handleCreateExpense = async (e: React.FormEvent) => {
        e.preventDefault();
        // TODO
    };

    // handle group list on page load
    const [groupList, setGroupList] = useState<GroupListItem[]>([]);

    useEffect(() => {
        const fetchGroupList = async () => {
            const response = await fetch(`${API_URL}/groups`, {
                method: "GET",
                credentials: "include",
            });

            const data: GroupListItem[] = await response.json();
            setGroupList(data);
        };

        fetchGroupList();
    }, []);

    // handle expense type on page load
    const [expenseTypes, setExpenseTypes] = useState<ExpenseTypeItem[]>([]);
    const [expTypeOptions, setExpTypeOptions] = useState<ReactElement[]>([]);

    useEffect(() => {
        const fetchExpeseTypes = async () => {
            const response = await fetch(`${API_URL}/expense_types`, {
                method: "GET",
                credentials: "include",
            });

            const data: ExpenseTypeItem[] = await response.json();
            setExpenseTypes(data);
        };

        fetchExpeseTypes();
    }, []);

    useEffect(() => {
        if (expenseTypes.length === 0) return;

        const options: ReactElement[] = [];
        let lastCategory = "";
        let generalId = "";
        expenseTypes.forEach((type) => {
            if (type.name === "General") {
                generalId = type.id;
            }
            if (lastCategory !== type.category) {
                options.push(
                    <option disabled key={type.category}>
                        ----- {type.category} -----
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
    }, [expenseTypes]);

    return (
        <div className="flex flex-row justify-center items-center py-5 w-screen">
            <form
                className="flex flex-col justify-center items-center py-5 space-y-5 md:w-1/3 w-5/6m max-w-md"
                onSubmit={handleCreateExpense}
            >
                <div className="text-2xl">Add Expense</div>

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
                <label className="input input-bordered flex items-center w-full">
                    <input
                        type="text"
                        id="description"
                        name="description"
                        className="grow"
                        placeholder="Description"
                    />
                </label>
                <div className="flex flex-row justify-start items-start w-full">
                    <select
                        className="select select-bordered w-1/3 text-base text-center"
                        id="currency"
                        name="currency"
                        defaultValue="CAD"
                    >
                        <option>CAD</option>
                        <option>NTD</option>
                        <option>USD</option>
                    </select>
                    <label className="input input-bordered flex items-center w-full">
                        <input
                            type="number"
                            id="total"
                            name="total"
                            className="grow"
                            step="0.001"
                            placeholder="0.00"
                        />
                    </label>
                </div>
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
                <SplitRule groupId={groupId || ""} />
                <button
                    id="submit"
                    type="submit"
                    className="btn btn-active btn-neutral btn-wide text-lg font-light"
                    disabled
                >
                    <Icon path={mdiCheckBold} size={1} />
                    OK
                </button>
                <div
                    id="indicator"
                    className={`${indicatorShow ? "" : "hidden"}`}
                >
                    <div className="flex justify-center items-center w-full">
                        <span className="loading loading-spinner loading-md"></span>
                    </div>
                </div>
                <div
                    id="feedback"
                    className={`${feedback.length === 0 ? "hidden" : ""}`}
                >
                    <div className="animate-fade">
                        <div role="alert" className="alert alert-success">
                            <Icon path={mdiCheckCircleOutline} size={1} />
                            <span>Your expense has been created!</span>
                        </div>
                    </div>
                </div>
            </form>
        </div>
    );
};

export default CreateExpense;
