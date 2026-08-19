#!/usr/bin/env python3
"""Regenerate the README hero GIF (demo/argus-demo.gif) from REAL command output.

Deterministic and browser-free: it actually runs `docker compose up -d` and
`python3 demo/drive.py`, captures their true output, lays it out as an asciicast v2
with typing/streaming timing, and renders it with `agg`.

    brew install agg        # one-time (asciinema -> gif, a single Rust binary)
    python3 demo/make-gif.py

Needs Docker running; leaves the zero-dependency stack up (compose down to stop it).
"""
import json, subprocess, sys, time

CAST, GIF = "demo/argus-demo.cast", "demo/argus-demo.gif"
W, H = 100, 30
PROMPT = "\x1b[38;5;39margus\x1b[0m \x1b[38;5;245m❯\x1b[0m "  # cyan 'argus' + grey chevron
CPS = 0.035   # seconds per typed char
events, clock = [], 0.0


def emit(dt, text):
    global clock
    clock += dt
    events.append([round(clock, 3), "o", text])


def type_cmd(cmd):
    emit(0.4, PROMPT)
    for ch in cmd:
        emit(CPS, ch)
    emit(0.35, "\r\n")


def stream(text, line_gap=0.12, first_gap=0.3):
    for i, ln in enumerate(text.rstrip("\n").split("\n")):
        emit(first_gap if i == 0 else line_gap, ln + "\r\n")


def run(cmd):
    p = subprocess.run(cmd, capture_output=True, text=True)
    return (p.stdout + p.stderr).rstrip("\n")


def wait_healthz(timeout=40):
    import urllib.request
    for _ in range(timeout):
        try:
            urllib.request.urlopen("http://localhost:8088/healthz", timeout=2)
            return
        except Exception:
            time.sleep(1)


# --- capture the real session ---
subprocess.run(["docker", "compose", "down", "--remove-orphans"], capture_output=True)
up = run(["docker", "compose", "up", "-d"])
wait_healthz()
drive = run([sys.executable, "demo/drive.py"])

# docker: keep only the settled states (Network Created + containers Started)
def keep(l):
    l = l.strip()
    return l.endswith("Started") or (l.startswith("Network") and l.endswith("Created"))
up_show = "\n".join("\x1b[38;5;245m" + l.strip() + "\x1b[0m"
                    for l in up.split("\n") if "argus" in l.lower() and keep(l))

# --- lay out and write the cast ---
emit(0.6, "\x1b[38;5;245m# Argus — catch an AI hallucination in the request path, "
          "in milliseconds\x1b[0m\r\n")
type_cmd("docker compose up -d")
stream(up_show, line_gap=0.10)
type_cmd("python3 demo/drive.py")
stream(drive, first_gap=0.4)
emit(1.0, PROMPT)   # land on a fresh prompt
emit(2.2, "")       # hold

header = {"version": 2, "width": W, "height": H,
          "env": {"TERM": "xterm-256color", "SHELL": "/bin/zsh"}}
with open(CAST, "w", encoding="utf-8") as f:
    f.write(json.dumps(header) + "\n")
    for ev in events:
        f.write(json.dumps(ev, ensure_ascii=False) + "\n")

# --- render ---
subprocess.run(["agg", "--theme", "dracula", "--font-size", "26",
                "--line-height", "1.35", "--fps-cap", "24", CAST, GIF], check=True)
print("wrote %s (%.1fs of tape)" % (GIF, clock))
