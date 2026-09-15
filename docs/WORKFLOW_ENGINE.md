# NusaMedia Workflow Engine

NusaMedia uses human-in-the-loop workflows for sensitive operations. A user request is not permission to mutate protected state.

## Wallet top-up
1. User submits amount, payment method and optional evidence.
2. System creates a `WALLET_TOPUP` case and Finance review task.
3. Finance receives workspace notifications; managers receive a real-time operational notification through the notification engine.
4. Reviewer records PASS/FAIL/ESCALATE with notes/evidence.
5. PASS moves the case to `AWAITING_APPROVAL`; wallet balance remains unchanged.
6. A separate Finance Manager or Admin Root approval is required. The reviewer cannot approve the same case (separation of duties).
7. Only final approval creates the wallet ledger transaction and credits balance.
8. Every transition is recorded in workflow events/audit trails.

## Identity verification
1. User submits verification type and document reference.
2. Trust & Safety receives a `DOCUMENT_REVIEW` task.
3. Reviewer checks document authenticity and consistency with the applicable authoritative verification process.
4. PASS moves to `AWAITING_MANAGER_APPROVAL`; FAIL rejects; ESCALATE creates an escalation state.
5. A separate Verification Manager or Admin Root must approve.
6. Only final approval marks the verification request approved and the user verified.

## Governance
- Admin Root is the highest authority and can manage roles, departments, workspaces, integrations and policies.
- Operational roles are scoped to their domain.
- Developer access does not imply financial, identity or production authority.
- Workspace notifications are scoped to active members; manager notifications are targeted separately.
- Sensitive actions use explicit state transitions rather than direct database mutations from UI actions.
- Evidence and decision notes are part of the case record.
