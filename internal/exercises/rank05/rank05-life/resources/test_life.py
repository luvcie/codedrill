import random
import subprocess


def expected(width, height, generations, script):
    alive = set()
    x = y = 0
    drawing = False
    for command in script:
        if command == "x":
            drawing = not drawing
        elif command == "w":
            y = max(0, y - 1)
        elif command == "s":
            y = min(height - 1, y + 1)
        elif command == "a":
            x = max(0, x - 1)
        elif command == "d":
            x = min(width - 1, x + 1)
        if drawing:
            alive.add((x, y))
    for _ in range(generations):
        counts = {}
        for x, y in alive:
            for dx in (-1, 0, 1):
                for dy in (-1, 0, 1):
                    if (dx or dy) and 0 <= x + dx < width and 0 <= y + dy < height:
                        cell = (x + dx, y + dy)
                        counts[cell] = counts.get(cell, 0) + 1
        alive = {cell for cell, count in counts.items()
                 if count == 3 or (count == 2 and cell in alive)}
    return "".join("".join("O" if (x, y) in alive else " "
                           for x in range(width)) + "\n"
                   for y in range(height)).encode()


rng = random.Random(504)
cases = [(1, 1, 0, "x"), (1, 1, 1, "x"), (4, 3, 0, ""),
         (4, 3, 0, "wwaaxddddssssaaaaxdd"),
         (5, 5, 1, "sdxss"), (5, 5, 2, "sdxss")]
for _ in range(30):
    cases.append((rng.randrange(1, 9), rng.randrange(1, 8), rng.randrange(6),
                  "".join(rng.choice("wasdx?\n") for _ in range(100))))
for width, height, generations, script in cases:
    result = subprocess.run(["./life", str(width), str(height), str(generations)],
                            input=script.encode(), capture_output=True, timeout=3)
    wanted = expected(width, height, generations, script)
    if result.returncode or result.stderr or result.stdout != wanted:
        raise SystemExit(f"life failed for {width}x{height}, {generations} generations, "
                         f"script {script!r}: expected {wanted!r}, got {result}")
print("life: all checks passed")
