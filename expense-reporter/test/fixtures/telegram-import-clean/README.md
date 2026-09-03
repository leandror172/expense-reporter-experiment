# telegram-import-clean

Two well-formed messages and nothing else. Exists to pin the T-63 rule on the converter's
rejects file: **when nothing was rejected, no `telegram-rejects.csv` is written** — its
existence is the signal. The other write scenarios use `telegram-import-dry-run`, which
always has rejects, so they cannot prove the absence.
