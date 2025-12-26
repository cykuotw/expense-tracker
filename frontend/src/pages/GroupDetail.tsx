import { Link } from "react-router-dom";
import { ExpenseData } from "../types/expense";
import ExpenseCard from "../components/expense/ExpanceCard";
import { GroupDetailProvider } from "../contexts/GroupDetailContext";
import { useGroupDetail } from "../hooks/GroupDetailContextHooks";

const GroupDetailContent = () => {
    const { groupinfo, balance, expenseList, loading, groupId, handleSettle } =
        useGroupDetail();

    if (loading) {
        return (
            <div className="flex justify-center items-center h-screen">
                <span className="loading loading-spinner loading-lg"></span>
            </div>
        );
    }

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
                                    <button
                                        className="btn btn-outline btn-error w-1/2"
                                        onClick={handleSettle}
                                    >
                                        SETTLE
                                    </button>
                                    <button className="btn w-1/2">
                                        Cancel
                                    </button>
                                </form>
                            </div>
                        </div>
                        <form method="dialog" className="modal-backdrop">
                            <button>close</button>
                        </form>
                    </dialog>
                </div>
                <div className="pt-3" id="unsettled-expenses">
                    {expenseList.length === 0
                        ? "No Expenses For Now"
                        : expenseList
                              .filter((exp: ExpenseData) => !exp.isSettled)
                              .map((exp: ExpenseData) => (
                                  <ExpenseCard key={exp.expenseId} {...exp} />
                              ))}
                </div>
                <div>
                    <div className="divider text-gray-500">
                        settled expenses
                    </div>
                    <div>
                        <div className="py-5">
                            {expenseList
                                .filter((exp: ExpenseData) => exp.isSettled)
                                .map((exp: ExpenseData) => (
                                    <ExpenseCard key={exp.expenseId} {...exp} />
                                ))}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default function GroupDetail() {
    return (
        <GroupDetailProvider>
            <GroupDetailContent />
        </GroupDetailProvider>
    );
}
