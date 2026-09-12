# Fixtures for `adapters/speckit` (issue 156)

| fixture | what it encodes |
|---|---|
| `authored/` | a written constitution with four `##` sections, and **two** spec directories — one numbered `001-add-auth`, one **not** numbered (`checkout-flow`), because 23.6% of real repositories do not number them |
| `template/` | the shipped constitution still carrying named placeholders. **10.2% of real constitutions are in this state**, and reading their articles as constraints would assert an architecture from a file nobody wrote (§37) |

`authored/specs/001-add-auth/tasks.md` carries a completed checkbox on purpose.
Nothing reads it: checkbox state is **execution** state (INV-002), and the
assertion that it never becomes a constraint or a decision needs a live one to
be worth anything.

There is no fixture with `specs/` and no `.specify/`. That case is not a Spec Kit
repository at all — measured, 50% of `specs/**/plan.md` matches carry no
`.specify/` — and detection must decline it.
