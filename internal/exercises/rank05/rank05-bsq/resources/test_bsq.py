from pathlib import Path
import random
import subprocess
import tempfile


def run(args, data, output, errors=b""):
    result = subprocess.run(["./bsq", *args], input=data, capture_output=True, timeout=3)
    if result.returncode or result.stdout != output or result.stderr != errors:
        raise SystemExit(f"bsq failed for args {args!r}, input {data!r}: "
                         f"expected {(output, errors)!r}, got {result}")


def solved(rows, empty=".", fill="x"):
    best = (0, 0, 0)
    for y in range(len(rows)):
        for x in range(len(rows[0])):
            for side in range(1, min(len(rows) - y, len(rows[0]) - x) + 1):
                if all(rows[r][c] == empty for r in range(y, y + side)
                       for c in range(x, x + side)) and side > best[0]:
                    best = (side, y, x)
    side, y, x = best
    result = [list(row) for row in rows]
    for r in range(y, y + side):
        for c in range(x, x + side):
            result[r][c] = fill
    return ("\n".join("".join(row) for row in result) + "\n").encode()


rng = random.Random(505)
for height, width in [(1, 1), (2, 5), (5, 2), (6, 6)]:
    for _ in range(8):
        rows = ["".join(rng.choice("...o") for _ in range(width)) for _ in range(height)]
        for header in (f"{height}.ox", f"{height} . o x"):
            data = (header + "\n" + "\n".join(rows) + "\n").encode()
            run([], data, solved(rows))
run([], b"2.ox\n.o\n..\n", b"xo\n..\n")
run([], b"2.ox\noo\noo\n", b"oo\noo\n")
run([], b"2.ox\n..\n..\n", b"xx\nxx\n")
run([], b"2123\n11\n11\n", b"33\n33\n")
run([], b"2 ox\n  \n  \n", b"xx\nxx\n")
for invalid in (b"", b"0.ox\n", b"1.oo\n.\n", b"1.ox\n\n", b"1.ox\n?\n",
                b"2.ox\n.\n..\n", b"2.ox\n.\n", b"1.ox\n.\n.\n",
                b"1.ox\n.", b"1.oxjunk\n.\n", b"1 . o\n.\n"):
    run([], invalid, b"", b"map error\n")
with tempfile.TemporaryDirectory() as directory:
    first, second = Path(directory) / "one", Path(directory) / "two"
    first.write_bytes(b"1.ox\n.\n")
    second.write_bytes(b"2.ox\n..\n..\n")
    missing = str(Path(directory) / "missing")
    run([str(first), str(second)], None, b"x\n\nxx\nxx\n")
    run([missing, str(first)], None, b"\nx\n", b"map error\n")
    run([str(first), missing], None, b"x\n\n", b"map error\n")
print("bsq: all checks passed")
