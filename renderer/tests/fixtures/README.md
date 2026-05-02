# Test fixtures

## `KATX_test.ar2v.gz`

NEXRAD Level II Archive volume scan from station KATX (Seattle), assembled
from real chunks at `s3://unidata-nexrad-level2-chunks/KATX/492/` on
2026-05-01. Chunks were concatenated in chunk-number order to produce a
Level II Archive blob byte-identical to a `_V06` archive file (the chunks
arrive in the already-bzip2-compressed Level II format that Py-ART reads
natively), then gzipped for repo storage.

This is a real volume scan checked in for deterministic decoder/renderer
tests. ~8.1 MB compressed (chunks are already bzip2-compressed internally,
so gzip provides minimal additional compression). Replace with a different
scan only if the current one stops being valid for the test assertions.

Original volume start timestamp: `20260501-180941` (UTC). Tests in
`test_decode.py` assert against this timestamp; bump them in lockstep
when replacing the fixture.

### Acquisition details

- Slot: 492
- Station: KATX
- Volume start timestamp: `20260501-180941` (2026-05-01T18:09:41Z)
- Chunk count: 67 (001-S through 067-E)
- All chunks share the single prefix `20260501-180941` (no Frankenstein risk)
- Final gzipped fixture size: 8,485,815 bytes (~8.1 MB)
- VCP: 35
- Lowest tilt elevation: ~0.483°

## Design deviation: no `chunks/` subdirectory

The original level2-renderer design (referenced in the comprehensive review
of `feat/level2-renderer-service`, GitHub issue #85, M10) called for a small
set of real bzip2-compressed chunks under `renderer/tests/fixtures/chunks/`
to cover `S3Client.download_volume` end-to-end without network.

The implementation instead uses:

- **Synthetic moto bytes** in `tests/test_s3.py` to verify chunk ordering,
  the AR2V/E suffix invariants, slot-overwrite filtering, and the parallel
  download path. Faster than network fixtures and exercises the same code
  paths.
- **`KATX_test.ar2v.gz`** in this directory for the assembled-volume
  byte-level decode/render path (`tests/test_decode.py`,
  `tests/test_render.py`, `tests/test_service.py`).

This is sufficient because `download_volume` is a pure parallel
concatenation — its only contract is "fetch every chunk, join in chunk-num
order" and the moto-backed tests cover both the happy path and the
missing-chunk error path. Adding real bzip2-framed chunk fixtures would
duplicate decode coverage already provided by the `_V06` archive fixture.

If a future change makes `download_volume` non-trivial (e.g. partial-volume
decoding before py-art, or streaming decompression at this layer), revisit
this decision and add a `chunks/` fixture set.
