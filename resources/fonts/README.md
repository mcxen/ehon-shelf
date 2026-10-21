# Fonts

Inter (weights 400, 500, 600, 700) and Noto Sans SC (weights 400, 500, 700) are static subsets, SIL OFL 1.1. The files match the subsets AvaMedia ships for its Apple-like UI. Licenses are the `OFL.txt` next to each family.

The app registers them as `Inter` and `Noto Sans SC`, and asks for `Inter, Noto Sans SC`. Inter is Latin only, so Simplified Chinese uses Noto Sans SC.

`subsets.json` records the AvaMedia subset sizes and checksums. `testdata/fonts/manifest.json` is what the checksum test reads.
