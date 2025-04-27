import { Link } from "react-router-dom";

import Icon from "@mdi/react";
import { mdiFoodForkDrink } from "@mdi/js";

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
                                        <Icon
                                            path={mdiFoodForkDrink}
                                            size={1}
                                        />
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
