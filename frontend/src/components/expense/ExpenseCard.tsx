import { Link } from "react-router-dom";
import Icon from "@mdi/react";
import { mdiFoodForkDrink } from "@mdi/js";
import { ExpenseData } from "../../types/expense";

export default function ExpenseCard(expense: ExpenseData) {
    const parsedDate = new Date(expense.expenseTime);
    const formattedDate = Number.isNaN(parsedDate.getTime())
        ? expense.expenseTime
        : new Intl.DateTimeFormat(undefined, {
              dateStyle: "medium",
              timeStyle: "short",
          }).format(parsedDate);

    return (
        <Link
            to={`/expense/${expense.expenseId}`}
            aria-label={`Open expense: ${expense.description}`}
            className="group flex flex-col gap-4 rounded-[1.5rem] border border-border bg-background p-4 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/60 hover:bg-primary/[0.04] hover:shadow-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 active:scale-[0.99] sm:flex-row sm:items-center sm:gap-4 sm:p-5"
        >
            <div className="flex size-11 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary transition-colors duration-200 group-hover:bg-primary group-hover:text-primary-foreground">
                <Icon path={mdiFoodForkDrink} size={1.25} />
            </div>
            <div className="min-w-0 flex-1">
                <div className="truncate text-base font-semibold tracking-[-0.01em] text-foreground">
                    {expense.description}
                </div>
                <div className="mt-1 text-sm text-foreground/65">
                    {formattedDate}
                </div>
            </div>
            <div className="flex items-center justify-between gap-3 border-t border-border/70 pt-3 sm:block sm:border-0 sm:pt-0 sm:text-right">
                <span className="text-xs font-semibold uppercase tracking-[0.16em] text-foreground/50 sm:hidden">
                    Total
                </span>
                <div className="text-lg font-semibold text-foreground transition-colors group-hover:text-primary">
                    ${expense.total} {expense.currency}
                </div>
                <div className="mt-1 hidden text-xs font-semibold uppercase tracking-[0.16em] text-primary sm:block">
                    View details
                </div>
            </div>
        </Link>
    );
}
