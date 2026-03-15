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
                className="panel-card block w-full rounded-[1.5rem] px-4 py-4 transition duration-200 hover:-translate-y-0.5 hover:shadow-md"
            >
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
                    <div className="flex items-center gap-3 sm:w-1/4">
                        <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                            <Icon path={mdiFoodForkDrink} size={0.9} />
                        </div>
                        <div className="text-xs font-semibold uppercase tracking-[0.2em] text-base-content/60">
                            {formatDate(expense.expenseTime)}
                        </div>
                    </div>

                    <div className="sm:w-2/4">
                        <div className="truncate text-base font-semibold tracking-[-0.02em]">
                            {expense.description}
                        </div>
                        <div className="mt-1 text-sm text-base-content/70">
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

                    <div className="sm:w-1/4 sm:text-right">
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
