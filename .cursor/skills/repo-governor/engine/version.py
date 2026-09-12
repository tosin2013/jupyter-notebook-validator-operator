"""The engine's version, in one place.

It was in two. `completion.py` held the constant; `envelope.py:213` restated
the same string as a literal. Both are written into decision records, which
ADR-009 requires for replay -- so a drift between them would mean two records
of the same engine disagreeing about which engine produced them, and the
divergence would be invisible until somebody tried to replay one.

Two sources of a derivable fact eventually disagree. This repository has now
recorded that sentence about ADR counts, about suite counts, and here.

`engine_min_version` in a manifest is a DIFFERENT fact -- the oldest engine
that can read that manifest -- and is deliberately not derived from this.
`tools/check-version.py` asserts the relationship rather than collapsing it.
"""

from __future__ import annotations

ENGINE_VERSION = "0.6.0"
