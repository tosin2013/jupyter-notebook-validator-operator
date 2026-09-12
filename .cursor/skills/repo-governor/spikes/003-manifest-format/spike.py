"""Spike for issue #3 — can a stdlib-only engine read the manifest as YAML?

Runs three checks:
  1. Does the subset parser produce the correct structure for the real manifest?
  2. Does it behave safely on adversarial / near-miss input?
  3. How does the equivalent JSON compare on size and reader complexity?
"""

import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import yaml_subset  # noqa: E402

HERE = Path(__file__).parent

EXPECTED = {
    "repo_governor": {"version": 1, "engine_min_version": "0.1.0"},
    "repository": {"id": "org/example-repo"},
    "condition": {"assessed": "L3", "profile": "GOVERNOR_FULL"},
    "providers": {
        "roadmap_authority": {
            "type": "linear",
            "project": "ENG",
            "contract_version": 1,
            "adapter": "adapters/linear",
        },
        "architecture": [
            {"type": "adr", "path": "docs/adr", "contract_version": 1},
            {"type": "openspec", "path": "openspec/"},
        ],
        "execution": {"type": "beads", "contract_version": 1},
        "repository": {"type": "git"},
        "change_signals": [{"type": "renovate"}],
        "retirement": [{"type": "repository_analysis"}],
        "decision_history": {"type": "builtin"},
    },
    "permissions": {
        "roadmap_authority": {"read": True, "write": False},
        "architecture": {"read": True, "write": False},
        "execution": {"read": True, "write": True},
    },
}

# (label, source, expectation) — "reject" means a YamlSubsetError is correct.
ADVERSARIAL = [
    ("tab indentation", "a:\n\tb: 1\n", "reject"),
    ("duplicate key", "a: 1\na: 2\n", "reject"),
    ("unterminated quote", 'a: "oops\n', "reject"),
    ("no colon", "just a line\n", "reject"),
    ("colon inside quoted value", 'a: "x: y"\n', {"a": "x: y"}),
    ("hash inside quoted value", 'a: "not # comment"\n', {"a": "not # comment"}),
    ("version-like string unquoted", "v: 0.1.0\n", {"v": "0.1.0"}),
    ("float coerced", "v: 1.5\n", {"v": 1.5}),
    ("norway problem", "country: no\n", {"country": False}),
    ("octal-looking string", "id: 0755\n", {"id": 755}),
    ("empty value is null", "a:\n", {"a": None}),
    ("flow mapping unsupported", "a: {b: 1}\n", {"a": "{b: 1}"}),
]


def check_roundtrip():
    text = (HERE / "manifest.yaml").read_text()
    got = yaml_subset.loads(text)
    ok = got == EXPECTED
    print(f"[1] subset parser matches expected structure : {'PASS' if ok else 'FAIL'}")
    if not ok:
        print("    expected:", json.dumps(EXPECTED, sort_keys=True)[:200])
        print("    got     :", json.dumps(got, sort_keys=True)[:200])
    return ok, got


def check_adversarial():
    print("[2] adversarial input")
    allok = True
    for label, src, expect in ADVERSARIAL:
        try:
            got = yaml_subset.loads(src)
            outcome = "reject" if False else got
        except yaml_subset.YamlSubsetError:
            outcome = "reject"
        except Exception as e:  # noqa: BLE001
            outcome = f"CRASH {type(e).__name__}"
        ok = outcome == expect
        allok &= ok
        note = "" if ok else f"   <-- expected {expect!r}, got {outcome!r}"
        print(f"    {'ok ' if ok else 'BAD'}  {label:<32} -> {str(outcome)[:34]}{note}")
    return allok


def check_json(parsed):
    jtext = json.dumps(parsed, indent=2) + "\n"
    (HERE / "manifest.json").write_text(jtext)
    ytext = (HERE / "manifest.yaml").read_text()
    ylines = len([l for l in ytext.splitlines() if l.strip() and not l.strip().startswith("#")])
    jlines = len(jtext.splitlines())
    parser_lines = len(
        [
            l
            for l in (HERE / "yaml_subset.py").read_text().splitlines()
            if l.strip() and not l.strip().startswith("#")
        ]
    )
    roundtrip = json.loads(jtext) == parsed
    print("[3] comparison")
    print(f"    YAML manifest lines (no comments)  : {ylines}")
    print(f"    JSON manifest lines                : {jlines}")
    print(f"    YAML subset parser lines (to own)  : {parser_lines}")
    print(f"    JSON reader lines (stdlib json)    : 1")
    print(f"    JSON round-trips exactly           : {roundtrip}")
    return roundtrip


# Values a real manifest could plausibly contain, and what they MUST mean.
# This is the check that decided issue #3.
HAZARDS = [
    ("engine_min_version: 1.0", "version string", "1.0"),
    ("contract_version: 1.0", "version string", "1.0"),
    ("assessed: L3", "level string", "L3"),
    ("type: no", "provider named 'no'", "no"),
    ("write: off", "permission bool", False),
    ("project: 0755", "project key", "0755"),
    ("id: 1e5", "id string", "1e5"),
    ("path: docs/adr", "path", "docs/adr"),
    ("adapter: {cmd: x}", "flow mapping must reject", "REJECT"),
    ("project: ON", "project key 'ON'", "ON"),
]


def check_hazards():
    """Not 'does the parser work' but 'is what it does SAFE'."""
    print("[4] hazard analysis — realistic manifest values")
    bad = 0
    for src, label, want in HAZARDS:
        try:
            got = list(yaml_subset.loads(src).values())[0]
        except yaml_subset.YamlSubsetError:
            got = "REJECT"
        ok = got == want
        bad += not ok
        print(f"    {'ok    ' if ok else 'HAZARD'}  {label:<24} {src:<24} -> {got!r:<12} want {want!r}")
    print(f"    => {bad}/{len(HAZARDS)} silently mis-typed")
    return bad


if __name__ == "__main__":
    ok1, parsed = check_roundtrip()
    ok2 = check_adversarial()
    ok3 = check_json(parsed if ok1 else EXPECTED)
    bad = check_hazards()
    print()
    print(f"SPIKE RESULT: parser is self-consistent ({'yes' if ok1 and ok2 and ok3 else 'no'}), "
          f"but {bad} realistic values are silently mis-typed.")
    print("CONCLUSION: a hand-rolled YAML subset trades 1 line of json.loads for 143 lines")
    print("            that produce silent wrong answers. See ADR-015.")
    sys.exit(0)
