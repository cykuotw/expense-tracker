import { Link } from "react-router-dom";
import Icon from "@mdi/react";
import { mdiFoodForkDrink } from "@mdi/js";
import { ExpenseData } from "../../types/expense";

export default function ExpenseCard(expense: ExpenseData) {
  return (
    <Link to={`/expenses/${expense.expenseId}`} className="expense-card">
      <div className="expense-card__icon">
        <Icon path={mdiFoodForkDrink} size={1.5} />
      </div>
      <div className="expense-card__info">
        <div className="expense-card__title">{expense.description}</div>
        <div className="expense-card__date">{expense.expenseTime}</div>
      </div>
      <div className="expense-card__amount">
        ${expense.total} {expense.currency}
      </div>
    </Link>
  );
}
