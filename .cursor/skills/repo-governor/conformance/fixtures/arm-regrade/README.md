# A saved arm, for re-grading

Three records with the transcripts that produced them, shaped exactly as
`--suite --out` writes them.

`--regrade` **rewrites the records in place** — that is its job. So nothing may
run it against this directory directly: a conformance suite that mutates a
tracked fixture leaves the tree dirty on a *passing* run, and this repository
has shipped that defect before. Copy it to a temporary directory first, which
is what `conformance/benchmark.py` and the acceptance criterion both do.

The `transcript` field inside each record deliberately points at a path that
does not exist, so anything reading it instead of the file beside the record
fails loudly rather than quietly.
