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
  model: default
  allowed-tools: get_balance update_state
---

Ask the user for the group ID if it is missing. Call get_balance for that group
before completing the task. Pass the result to finish_task when done.
Preserve exact amount strings and original currencies; do not net currencies or
calculate new balances. If a tool fails, report that the balance is unavailable.
Never request credentials or mutate financial data. Do not include internal reasoning.

Use update_state to remember an explicitly supplied group ID as selectedGroupId
when useful for follow-up turns. Other task context may use meaningful JSON keys
chosen for the conversation; no namespace is required. set replaces whole
top-level values, unset removes keys, and omitted keys remain. Saved context
never proves group access or current balances: still call get_balance for a fresh
authorized result before completion. Store no credentials or financial results.
The state tool changes session context only; it does not mutate expenses.
Call update_state alone, never alongside another tool in the same response.
Combine related changes into one set/unset call.
