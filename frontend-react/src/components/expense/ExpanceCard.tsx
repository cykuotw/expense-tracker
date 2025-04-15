import { Link } from "react-router-dom";
import { ExpenseData } from "../../types/expense";

export default function ExpenseCard(expense: ExpenseData) {
    const formatDate = (timestamp: string): string => {
        const date = new Date(timestamp);
        const month = date.toLocaleString("en-US", { month: "short" });
        const year = date.getFullYear();
        return `${year} ${month}`;
    };

    return (
        <div className="flex flex-col">
            <div className="grow h-20 w-screen py-1 px-5">
                <div className="flex flex-row justify-center items-center h-full">
                    <div className="h-full w-11/12 md:w-7/12 border border-dotted rounded-xl max-w-md">
                        <Link
                            to={`/expense/${expense.expenseId}`}
                            className="flex justify-center items-center h-full w-full"
                        >
                            <div className="flex flex-row justify-center items-center w-full">
                                <div className="flex flex-col justify-center items-center w-2/12 mx-1">
                                    <div className="flex-none text-sm">
                                        <p>{formatDate(expense.expenseTime)}</p>
                                    </div>
                                </div>
                                <div className="flex flex-col justify-center items-center w-1/12 mx-1">
                                    <div className="flex-none">
                                        <svg
                                            xmlns="http://www.w3.org/2000/svg"
                                            width="24"
                                            height="24"
                                            viewBox="0 0 24 24"
                                        >
                                            <title>food-fork-drink</title>
                                            <path d="M3,3A1,1 0 0,0 2,4V8L2,9.5C2,11.19 3.03,12.63 4.5,13.22V19.5A1.5,1.5 0 0,0 6,21A1.5,1.5 0 0,0 7.5,19.5V13.22C8.97,12.63 10,11.19 10,9.5V8L10,4A1,1 0 0,0 9,3A1,1 0 0,0 8,4V8A0.5,0.5 0 0,1 7.5,8.5A0.5,0.5 0 0,1 7,8V4A1,1 0 0,0 6,3A1,1 0 0,0 5,4V8A0.5,0.5 0 0,1 4.5,8.5A0.5,0.5 0 0,1 4,8V4A1,1 0 0,0 3,3M19.88,3C19.75,3 19.62,3.09 19.5,3.16L16,5.25V9H12V11H13L14,21H20L21,11H22V9H18V6.34L20.5,4.84C21,4.56 21.13,4 20.84,3.5C20.63,3.14 20.26,2.95 19.88,3Z"></path>
                                        </svg>
                                    </div>
                                </div>
                                <div className="flex flex-col justify-center w-6/12 truncate overflow-hidden mx-1">
                                    <div className="flex-none w-fit">
                                        <p className="">
                                            {expense.description}
                                        </p>
                                    </div>
                                    <div className="flex-none w-fit">
                                        {expense.payerUserIds.map((id, i) => {
                                            if (id === expense.currentUser) {
                                                return (
                                                    <p
                                                        key={id}
                                                        className="text-xs"
                                                    >
                                                        {`You paid $${expense.total} ${expense.currency}`}
                                                    </p>
                                                );
                                            } else {
                                                return (
                                                    <p
                                                        key={id}
                                                        className="text-xs"
                                                    >
                                                        {`${expense.payerUsernames[i]} paid $${expense.total} ${expense.currency}`}
                                                    </p>
                                                );
                                            }
                                        })}
                                    </div>
                                </div>
                                <div className="flex flex-col justify-center items-center w-2/6">
                                    <div className="flex-none w-fit text-sm">
                                        {expense.payerUserIds[0] ===
                                        expense.currentUser ? (
                                            <p>{`You lend $${expense.total} ${expense.currency}`}</p>
                                        ) : (
                                            <p>{`You owe $${expense.total} ${expense.currency}`}</p>
                                        )}
                                    </div>
                                </div>
                            </div>
                        </Link>
                    </div>
                </div>
            </div>
        </div>
    );
}
