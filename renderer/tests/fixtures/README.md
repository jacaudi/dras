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
