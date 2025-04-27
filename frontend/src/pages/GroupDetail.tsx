import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";

import { GroupInfo } from "../types/group";
import { BalanceData } from "../types/balance";
import { ExpenseData } from "../types/expense";
import ExpenseCard from "../components/expense/ExpanceCard";
import { API_URL } from "../configs/config";

export default function GroupDetail() {
    const [groupinfo, setGroupInfo] = useState<GroupInfo | null>(null);
    const [balance, setBalance] = useState<BalanceData | null>(null);
    const [expenseList, setExpenseList] = useState<ExpenseData[]>([]);

    const { id: groupId } = useParams();

    useEffect(() => {
        const fetchGroupInfo = async () => {
            try {
                const response = await fetch(`${API_URL}/group/${groupId}`, {
                    method: "GET",
                    credentials: "include",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                const data = await response.json();
                setGroupInfo(data);
            } catch (error) {
                console.log(error);
            }
        };

        const fetchBalance = async () => {
            try {
                const response = await fetch(`${API_URL}/balance/${groupId}`, {
                    method: "GET",
                    credentials: "include",
                    headers: {
                        "Content-Type": "application/json",
                    },
                });
                const data = await response.json();
                setBalance(data);
            } catch (error) {
                console.log(error);
            }
        };

        const fetchExpenses = async () => {
            try {
                const response = await fetch(
                    `${API_URL}/expense_list/${groupId}`,
                    {
                        method: "GET",
                        credentials: "include",
                        headers: {
                            "Content-Type": "application/json",
                        },
                    }
                );
                const data = await response.json();
                setExpenseList(data);
            } catch (error) {
                console.log(error);
            }
        };

        fetchGroupInfo();
        fetchBalance();
        fetchExpenses();
    }, [groupId]);

    return (
        <div className="flex justify-center items-center py-5">
            <div className="text-center">
                <h1 className="text-3xl font-semibold text-success">
                    {groupinfo?.groupName}
                </h1>
                <div className="flex flex-col items-center py-5">
                    {!balance?.balances || balance.balances.length === 0 ? (
                        <p className="py-5">All Balanced!</p>
                    ) : (
                        balance.balances.map((b) => {
                            if (b.receiverUserId == balance.currentUser) {
                                return (
                                    <p key={b.id}>
                                        {b.senderUsername} owes you $
                                        {b.balance + " " + balance.currency}
                                    </p>
                                );
                            }
                            if (b.senderUserId == balance.currentUser) {
                                return (
                                    <p key={b.id}>
                                        You owe {b.receiverUsername} $
                                        {b.balance + " " + balance.currency}
                                    </p>
                                );
                            }
                        })
                    )}
                </div>
                <div className="flex flex-col items-center space-y-1">
                    <button className="btn btn-wide btn-outline btn-success py-5 font-bold">
                        <Link to={`/create_expense?g=${groupId}`}>
                            ADD EXPENSE
                        </Link>
                    </button>
                    <button className="btn btn-wide btn-outline btn-info py-5 font-bold">
                        <Link to={`/add_member?g=${groupId}`}>
                            ADD MEMEBERS
                        </Link>
                    </button>
                    <button
                        className="btn btn-wide btn-outline btn-error py-5 font-bold"
                        onClick={() => {
                            const dialog = document.getElementById(
                                "settle_confirm"
                            ) as HTMLDialogElement | null;
                            dialog?.showModal();
                        }}
                    >
                        SETTLE UP
                    </button>
                    <dialog id="settle_confirm" className="modal">
                        <div className="modal-box">
                            <h3 className="text-lg font-bold">Are You Sure?</h3>
                            <p className="py-4">
                                Press SETTLE to settle expenses.
                            </p>
                            <div className="modal-action">
                                <form
                                    method="dialog"
                                    className="flex space-x-1"
                                >
                                    <button className="btn btn-outline btn-error w-1/2">
                                        SETTLE
                                    </button>
                                    <button className="btn w-1/2">
                                        Cancel
                                    </button>
                                </form>
                            </div>
                        </div>
                    </dialog>
                </div>
                <div className="pt-3" id="unsettled-expenses">
                    {expenseList.length === 0
                        ? "No Expenses For Now"
                        : expenseList.map((exp: ExpenseData) => (
                              <ExpenseCard key={exp.expenseId} {...exp} />
                          ))}
                </div>
                {/* <div id="settled-expenses">settled expenses</div> */}
                <div>
                    <div className="py-5">
                        {/* {
                        expenseList.length !== 0 ? (
                            <hr className="block md:hidden"/>
                            <button className="my-2 btn btn-ghost">More Settled Expenses</button>
                        )
                    } */}
                    </div>
                </div>
            </div>
        </div>
    );
}
