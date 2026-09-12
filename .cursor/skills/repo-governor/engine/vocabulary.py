#!/usr/bin/env python3
"""Closed vocabularies and the blocking classifier — gate 7 (ADR-007).

Three closed sets, and the rule that decides whether an unknown stops work.

Closed means closed: an unrecognised value is a defect, not an extension
point. ADR-007 rule 1 makes the disposition alphabet engine-owned precisely
so cross-provider portability (§53) is measurable — two providers can only
be compared if the output alphabet is fixed. The same argument applies to
unknown reasons, or a provider could invent a reason the engine cannot
classify as blocking or not.

The reason set below was harvested from what the adapters and engine
actually emit, then closed around it. Adding a reason requires touching
this file, which is the point.
"""

from __future__ import annotations

# --- §41 governance dispositions ------------------------------------------
EXECUTION = ("EXECUTE", "CONTINUE", "STOP_COMPLETE")
REVIEW = ("CAPTURE_ONLY", "ROADMAP_REVIEW", "ARCHITECTURE_REVIEW",
          "MAINTENANCE_REVIEW", "RETIREMENT_REVIEW")
REFUSAL = ("NO_EXECUTION_AUTHORITY", "AUTHORITY_WITHDRAWN", "CONFLICT", "UNKNOWN")
DISPOSITIONS = EXECUTION + REVIEW + REFUSAL

# --- §42 onboarding dispositions — a SEPARATE alphabet --------------------
# Different state machine, different consumer (a human running onboarding,
# not an agent mid-task). These never appear in a governance decision.
ONBOARDING = ("PROVIDER_DETECTED", "PROVIDER_UNCONFIRMED", "PROVIDER_CONFIGURED",
              "PROVIDER_UNAVAILABLE", "PROVIDER_CONFLICT", "AUTHORITY_SOURCE_MISSING",
              "READY_FOR_GOVERNANCE", "PROPOSAL_READY")

# --- typed adapter errors (ADR-003) ---------------------------------------
ERRORS = ("PROVIDER_UNAVAILABLE", "NOT_FOUND", "UNSUPPORTED_FUNCTION",
          "MALFORMED_SOURCE", "BAD_REQUEST")

# --- unknown reasons, with the dimension each belongs to and whether it blocks
#
# BLOCKING means: this unknown sits on the critical path to EXECUTE, so the
# engine must not proceed. NON-BLOCKING means it is real uncertainty that
# does not gate the current decision. That split is what keeps INV-012 from
# colliding with §54's "turns all discoveries into human review" failure
# condition (ADR-007 rule 4).
#
# The distinction is NOT "how serious is this" — it is "does the current
# decision depend on it".
REASONS = {
    # authority — always blocking: without admission there is no authorization
    "AUTHORITY_UNSTATED":            ("authority", True,
                                      "The work item declares no authority value."),
    "NOT_ADMITTED":                  ("authority", True,
                                      "Filed but never admitted to the roadmap (e.g. Linear triage)."),
    "ADMISSION_SIGNAL_UNDECLARED":    ("authority", True,
                                      "The binding declares no admission signal; GitHub expresses "
                                      "admission by convention and the adapter must not choose one."),
    "NOT_ON_BOARD":                  ("authority", True,
                                      "Not an item on any Project, so admission cannot be read."),

    # acceptance — non-blocking: absence means no completion bar, not danger
    "NO_CRITERIA_DECLARED":          ("acceptance", False,
                                      "No acceptance criteria declared; STOP_COMPLETE unavailable."),
    # A bar that declares it covers only part of its authority. Non-blocking:
    # the covered part genuinely passed, and the decision is already CONTINUE
    # -- this explains WHY completion is unavailable when every criterion is
    # green, which is otherwise the most confusing output the engine can emit.
    "BAR_COVERS_PART":               ("evidence", False,
                                      "The declared bar is satisfied but covers only part of "
                                      "this item; STOP_COMPLETE unavailable."),
    # The other side of BAR_COVERS_PART. A bar that named where its uncovered
    # half went, and every named authority is complete, HAS established the
    # whole item -- so this is recorded rather than merely absent, because
    # "completed with a discharged split" and "completed with nothing to split"
    # are different facts about how a decision was reached (issue 171).
    "BAR_COVERS_PART_DISCHARGED":    ("evidence", False,
                                      "The bar covered part of this item and the rest was "
                                      "split into authorities that are themselves complete."),
    "ACCEPTANCE_UNSTATED":           ("acceptance", False,
                                      "Provider cannot supply machine-checkable acceptance conditions."),
    "NON_GOALS_UNSTATED":            ("scope", False,
                                      "No non-goals declared; the scope envelope will be thin."),
    "PARENT_NOT_RESOLVABLE_ON_TRANSPORT": ("scope", False,
                                      "The bound transport can name the parent only by opaque id. "
                                      "Capabilities belong to (provider x transport), so this is an "
                                      "honest absence on this transport, not an absence of parent."),
    "SCOPE_NOT_STRUCTURED":          ("scope", False,
                                      "Scope exists only as free prose, which may not become typed facts."),

    # architecture — non-blocking: UNKNOWN architecture constrains nothing
    "NO_ARCHITECTURE_EVIDENCE":      ("architecture", False,
                                      "No architecture decisions found."),
    "NO_ACCEPTED_DECISIONS":         ("architecture", False,
                                      "Decisions exist but none is Accepted; state is INFERRED, not DEFINED."),
    "STATUS_UNSTATED":               ("architecture", False,
                                      "Decision files carry no Status line. A repository that has "
                                      "not declared statuses is not blocked from working; it is "
                                      "simply not constrained by decisions it never accepted."),
    "STATUS_TEMPLATE_UNFILLED":      ("architecture", True,
                                      "A decision file carries an unfilled template placeholder "
                                      "instead of a status. It used to fall out of every bucket "
                                      "silently -- neither active, nor historical, nor unstated."),
    "STATUS_DIALECTS_DISAGREE":      ("architecture", True,
                                      "Two status dialects in one file disagree. Precedence would "
                                      "pick a winner where the document genuinely says two things."),
    "ARCHITECTURE_PARTIALLY_READ":   ("architecture", True,
                                      "More decision files could not be read than constraints were "
                                      "established from, so the constraint set is dominated by what "
                                      "was not read. Governing on the readable minority would state "
                                      "an architecture that is mostly unexamined."),

    # decision history — absence is not permission, but it does not block either
    "NO_DECISION_RECORDED":          ("decision_history", False,
                                      "No prior decision recorded; the question was never asked. "
                                      "Absence of a decision is not permission (INV-005)."),
    # execution — non-blocking: execution state never authorizes (INV-002)
    "NO_HANDOFF_RECORDED":           ("execution", False,
                                      "No handoff state; agent continuity is not recoverable."),
    "HISTORY_NOT_RETAINED":          ("execution", False,
                                      "Provider stores current state only, not an append-only history."),

    # evidence — blocking: a check that cannot be evaluated cannot be claimed
    "CHECK_TIMED_OUT":               ("evidence", True,
                                      "An acceptance check did not finish."),
    "NOT_DERIVABLE_FROM_SOURCE":     ("evidence", False,
                                      "Requires analysis the bound provider cannot perform."),

    # retirement — blocking: unresolved obligations must never permit removal
    "NOT_VISIBLE_TO_STATIC_ANALYSIS": ("retirement", True,
                                       "Dynamic loading, runtime usage or public contracts are invisible to grep."),
    "NO_RETIREMENT_EVIDENCE":        ("retirement", True,
                                      "No retirement provider is bound, so nothing was checked. "
                                      "Nothing checked must never read as nothing owed."),

    # change signals — non-blocking: a signal is not work (INV-006)
    "IMPACT_NOT_ASSESSED":           ("change_signals", False,
                                      "Whether this change matters here is a judgement, not a supplied fact."),

    # transport / provider health — blocking
    "PROVIDER_UNREACHABLE":          ("provider", True,
                                      "The bound provider could not be reached."),
    "TRANSPORT_UNCONFIGURED":        ("provider", True,
                                      "No transport is configured, so the provider advertises nothing."),
    "MANIFEST_UNREADABLE":           ("provider", True,
                                      "The governance manifest is missing or invalid for the repository "
                                      "being governed, so no binding is known and nothing may be ruled."),
}

DIMENSIONS = ("authority", "acceptance", "scope", "architecture", "execution",
              "evidence", "retirement", "change_signals", "provider", "decision_history")

# Profiles may only make a non-blocking reason blocking, never the reverse.
# Loosening a blocking reason would let a profile permit EXECUTE on evidence
# the engine could not resolve.
class VocabularyError(ValueError):
    pass


def _load_profiles():
    """Escalations come from policies/*.json, not from a literal here.

    ADR-006 rule 4: profiles are declarative policy packs loaded on demand,
    which is what lets an L1 repository never pay for L4 policy detail.
    JSON rather than the YAML ADR-001 sketched, because ADR-011 leaves the
    engine with no YAML parser and ADR-015 made JSON canonical.
    """
    import json
    from pathlib import Path
    out = {}
    d = Path(__file__).resolve().parent.parent / "policies"
    if not d.is_dir():
        return out
    for p in sorted(d.glob("*.json")):
        try:
            pol = json.loads(p.read_text())
        except json.JSONDecodeError as e:
            raise VocabularyError(f"{p} is not valid JSON: {e}") from None
        out[pol["profile"]] = tuple(pol.get("escalate_to_blocking", []))
    return out


def _load_required_roles():
    """Which provider roles a profile requires (ADR-006 rule 4).

    The mapping has been in policies/*.json since the profiles were written and
    was read by nothing -- ADR-006 line 18 says the profile "determines which
    policy packs load and which providers are required", and only the first
    half was ever implemented. A repository assessed L4 loaded a profile
    demanding five roles, bound two, and nothing said a word (issue 79).
    """
    import json
    from pathlib import Path
    out = {}
    d = Path(__file__).resolve().parent.parent / "policies"
    if not d.is_dir():
        return out
    for p in sorted(d.glob("*.json")):
        pol = json.loads(p.read_text())
        out[pol["profile"]] = tuple(pol.get("required_roles", []))
    return out


PROFILE_ESCALATIONS = _load_profiles()
PROFILE_REQUIRED_ROLES = _load_required_roles()


def required_roles(profile):
    """Roles this profile requires. Unknown profile => empty, never a guess.

    An unrecognised profile must not silently require nothing AND must not
    invent a default: callers distinguish the two by asking whether the
    profile is known, which is why this does not raise.
    """
    return PROFILE_REQUIRED_ROLES.get(profile, ())


def classify(reason, profile="GOVERNOR_LITE"):
    """Return (dimension, blocking, description). Raises on an unknown reason."""
    if reason not in REASONS:
        raise VocabularyError(
            f"{reason!r} is not in the closed reason vocabulary. "
            "Adding one requires editing engine/vocabulary.py — a provider may not invent reasons.")
    dim, blocking, desc = REASONS[reason]
    if not blocking and reason in PROFILE_ESCALATIONS.get(profile, ()):
        return dim, True, desc + f" (escalated to blocking by {profile})"
    return dim, blocking, desc


def is_disposition(value):
    return value in DISPOSITIONS


def check_alphabets_disjoint():
    """Governance and onboarding alphabets must not overlap (ADR-007 rule 6).

    PROVIDER_UNAVAILABLE is deliberately shared with the ERRORS set: it is an
    adapter-level error type AND an onboarding disposition. It is not a
    governance disposition, which is the separation that matters.
    """
    overlap = set(DISPOSITIONS) & set(ONBOARDING)
    if overlap:
        raise VocabularyError(f"governance and onboarding alphabets overlap: {sorted(overlap)}")
    return True


if __name__ == "__main__":
    check_alphabets_disjoint()
    print(f"governance dispositions : {len(DISPOSITIONS)}")
    print(f"onboarding dispositions : {len(ONBOARDING)}")
    print(f"adapter error types     : {len(ERRORS)}")
    print(f"unknown reasons         : {len(REASONS)}")
    print(f"dimensions              : {len(DIMENSIONS)}")
    blocking = [r for r, (_, b, _) in REASONS.items() if b]
    print(f"  blocking              : {len(blocking)}")
    print(f"  non-blocking          : {len(REASONS) - len(blocking)}")
