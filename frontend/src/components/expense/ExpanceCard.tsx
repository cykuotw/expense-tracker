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
        <div className="w-full">
            <Link
                to={`/expense/${expense.expenseId}`}
                className="block w-full rounded-2xl border border-base-200 bg-base-100/90 px-4 py-3 shadow-sm transition duration-200 hover:-translate-y-0.5 hover:shadow-md"
            >
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
                    <div className="flex items-center gap-3 sm:w-1/5">
                        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-base-200 text-base-content/70">
                            <Icon path={mdiFoodForkDrink} size={0.9} />
                        </div>
                        <div className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                            {formatDate(expense.expenseTime)}
                        </div>
                    </div>

                    <div className="sm:w-3/5">
                        <div className="truncate text-base font-semibold">
                            {expense.description}
                        </div>
                        <div className="mt-1 text-xs text-base-content/70">
                            {expense.payerUserIds.map((id, i) => {
                                if (id === expense.currentUser) {
                                    return (
                                        <span key={id}>
                                            You paid ${expense.total}{" "}
                                            {expense.currency}
                                        </span>
                                    );
                                }
                                return (
                                    <span key={id}>
                                        {expense.payerUsernames[i]} paid $
                                        {expense.total} {expense.currency}
                                    </span>
                                );
                            })}
                        </div>
                    </div>

                    <div className="sm:w-1/5 sm:text-right">
                        <div className="text-sm font-semibold text-base-content">
                            {expense.payerUserIds[0] === expense.currentUser
                                ? `You lend $${expense.total} ${expense.currency}`
                                : `You owe $${expense.total} ${expense.currency}`}
                        </div>
                    </div>
                </div>
            </Link>
        </div>
    );
}
