# Relationship constraints

This matrix records the database-enforced relationships introduced by migrations
`000024` and `000034`. Application authorization remains responsible for proving
that expense participants belong to the affected group.

| Child relationship | Parent | Delete behavior | Additional constraint |
| --- | --- | --- | --- |
| `groups.create_by_user_id` | `users.id` | `RESTRICT` | — |
| `group_member.group_id` | `groups.id` | `CASCADE` | `(group_id, user_id)` is unique |
| `group_member.user_id` | `users.id` | `RESTRICT` | `(group_id, user_id)` is unique |
| `expense.group_id` | `groups.id` | `RESTRICT` | — |
| `expense.create_by_user_id` | `users.id` | `RESTRICT` | — |
| `expense.pay_by_user_id` | `users.id` | `RESTRICT` | — |
| `expense.exp_type_id` | `expense_type.id` | `RESTRICT` | — |
| `item.expense_id` | `expense.id` | `CASCADE` | — |
| `ledger.expense_id` | `expense.id` | `CASCADE` | — |
| `ledger.lender_user_id` | `users.id` | `RESTRICT` | — |
| `ledger.borrower_user_id` | `users.id` | `RESTRICT` | — |
| `balance.group_id` | `groups.id` | `CASCADE` | — |
| `balance.sender_user_id` | `users.id` | `RESTRICT` | — |
| `balance.receiver_user_id` | `users.id` | `RESTRICT` | — |
| `balance_ledger.balance_id` | `balance.id` | `CASCADE` | `(balance_id, ledger_id)` is unique |
| `balance_ledger.ledger_id` | `ledger.id` | `CASCADE` | `(balance_id, ledger_id)` is unique |

Expense deletion is a soft delete and does not invoke these cascades. The
`CASCADE` rules apply only when a parent row is physically deleted. User and
expense ownership relationships use `RESTRICT` where deleting the parent would
erase or orphan business history.

## Migration 000034 preflight

Run this read-only query before applying migration `000034`:

```sql
SELECT balance_id, ledger_id, COUNT(*) AS association_count
FROM balance_ledger
GROUP BY balance_id, ledger_id
HAVING COUNT(*) > 1
ORDER BY balance_id, ledger_id;
```

An empty result is safe to migrate. If the query returns rows, investigate each
association as business data and resolve it deliberately. Migration `000034`
runs the same check while holding a table lock and aborts without deleting or
rewriting rows when duplicates remain.
