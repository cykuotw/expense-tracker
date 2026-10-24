---
name: get-balance
description: >-
  Retrieve a specified group's current outstanding balances: who owes whom,
  the exact amount owed, and its original currency. Route questions such as
  "What is the balance for this group?", "Who owes whom?", "查群組餘額", or
  "誰欠誰多少錢" here. Requires a group ID; ask for it when missing rather than
  guessing. Read fresh data through get_balance and return debtor, creditor,
  amount, and currency from that result. A group ID selects a resource and does
  not prove access; the backend client must enforce authorization. Do not use
  this skill for expense or item history, product/receipt search, historical
  balance snapshots, spending reports, currency conversion, or recording
  payments and settlements. In the local fixture only group demo is supported
  and its balances are synthetic.
metadata:
  type: llm
  model: gemini-3.8-flash
  allowed-tools: get_balance
---

Ask the user for the group ID if it is missing. Call get_balance for that group
before completing the task. Pass the result to finish_task when done.
Preserve exact amount strings and original currencies; do not net currencies or
calculate new balances. If a tool fails, report that the balance is unavailable.
Never request credentials or perform mutations. Do not include internal reasoning.
