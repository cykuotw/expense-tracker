import { Link } from "react-router-dom";
import Icon from "@mdi/react";
import { ExpenseData } from "../../types/expense";
import { getExpenseTypePresentation } from "../../lib/expenseCategoryPresentation";

export default function ExpenseCard(expense: ExpenseData) {
    const parsedDate = new Date(expense.expenseTime);
    const validDate = !Number.isNaN(parsedDate.getTime());
    const dateMonth = validDate
        ? new Intl.DateTimeFormat("en-US", { month: "short" }).format(
              parsedDate
          )
        : "";
    const dateDay = validDate
        ? new Intl.DateTimeFormat(undefined, { day: "numeric" }).format(
              parsedDate
          )
        : expense.expenseTime;
    const presentation = getExpenseTypePresentation(
        expense.expenseCategory,
        expense.expenseType
    );

    return (
        <Link
            to={`/expense/${expense.expenseId}`}
            aria-label={`Open expense: ${expense.description}`}
            className="group grid grid-cols-[2.25rem_2.75rem_minmax(0,1fr)] items-center gap-x-2.5 gap-y-2 rounded-[1.5rem] border border-border bg-background p-4 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/60 hover:bg-primary/[0.04] hover:shadow-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 active:scale-[0.99] md:grid-cols-[2.75rem_3.25rem_minmax(0,1fr)_auto] md:gap-x-3 md:p-5"
        >
            <time
                dateTime={validDate ? parsedDate.toISOString() : undefined}
                className="row-span-2 flex flex-col text-center leading-none text-foreground/60"
            >
                {dateMonth && (
                    <span className="text-[0.65rem] font-semibold uppercase tracking-[0.12em]">
                        {dateMonth}
                    </span>
                )}
                <span className="mt-1 text-xl font-medium tracking-[-0.04em] text-foreground/80 md:text-2xl">
                    {dateDay}
                </span>
            </time>
            <div
                className={`row-span-2 flex size-11 shrink-0 items-center justify-center rounded-xl transition-colors duration-200 md:size-13 ${presentation.iconClassName}`}
            >
                <Icon path={presentation.icon} size={1.2} />
            </div>
            <div className="min-w-0 self-center">
                <div className="break-words text-lg font-semibold leading-tight tracking-[-0.01em] text-foreground md:text-xl">
                    {expense.description || expense.expenseType}
                </div>
                <div className="mt-1 truncate text-sm text-foreground/65 md:text-base">
                    {expense.payerUsernames[0]
                        ? `Paid by ${expense.payerUsernames[0]}`
                        : expense.expenseType}
                </div>
            </div>
            <div className="col-start-3 flex items-baseline justify-between gap-3 border-t border-border/70 pt-3 md:col-start-4 md:row-span-2 md:block md:border-0 md:pt-0 md:text-right">
                <span className="text-xs font-semibold uppercase tracking-[0.16em] text-foreground/50">
                    Total
                </span>
                <div className="text-xl font-semibold tabular-nums text-foreground transition-colors group-hover:text-primary md:mt-1 md:text-2xl">
                    ${expense.total} {expense.currency}
                </div>
            </div>
        </Link>
    );
}
