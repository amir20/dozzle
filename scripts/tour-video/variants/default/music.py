import sys
import numpy as np
from scipy.signal import butter, sosfilt
from scipy.io import wavfile

SR = 44100
DUR = 33.0
N = int(SR * DUR)
t = np.arange(N) / SR
rng = np.random.default_rng(7)
BEAT = 0.5  # 120 bpm
DROP = 4.6
OUT = 29.0


def hz(m):
    return 440.0 * 2 ** ((m - 69) / 12)


def lp(x, f, o=2):
    return sosfilt(butter(o, f, "low", fs=SR, output="sos"), x)


def hp(x, f, o=2):
    return sosfilt(butter(o, f, "high", fs=SR, output="sos"), x)


def bp(x, lo, hi, o=2):
    return sosfilt(butter(o, [lo, hi], "band", fs=SR, output="sos"), x)


def place(buf, sig, at, gain=1.0):
    i = int(at * SR)
    if i >= len(buf):
        return
    j = min(len(buf), i + len(sig))
    buf[i:j] += sig[: j - i] * gain


def env(n, a, d):
    e = np.ones(n)
    na = max(1, int(a * SR))
    e[:na] = np.linspace(0, 1, na)
    e *= np.exp(-np.arange(n) / SR / d)
    return e


L = np.zeros(N)
R = np.zeros(N)


def add(sig, at, gain=1.0, pan=0.0):
    place(L, sig, at, gain * np.sqrt(0.5 * (1 - pan)))
    place(R, sig, at, gain * np.sqrt(0.5 * (1 + pan)))


# chords (A minor): Am F C G, one per bar (2s)
CHORDS = [[57, 60, 64, 69], [53, 57, 60, 65], [55, 60, 64, 67], [55, 59, 62, 67]]
ROOTS = [45, 41, 48, 43]

# --- pad: detuned saws, dark before the drop, opens after
pad = np.zeros(N)
for bar in range(int(DUR // 2) + 1):
    ch = CHORDS[bar % 4]
    s, e = bar * 2.0, bar * 2.0 + 2.0
    seg = (t >= s) & (t < e + 0.3)
    tt = t[seg] - s
    sig = np.zeros(seg.sum())
    for m in ch:
        for det in (-0.07, 0.07):
            f = hz(m + det)
            ph = (f * tt) % 1.0
            sig += (2 * ph - 1) * 0.12
    fade = np.clip(tt / 0.08, 0, 1) * np.clip((2.3 - tt) / 0.3, 0, 1)
    pad[seg] += sig * fade
cut_dark = lp(pad, 700, 2)
cut_open = lp(pad, 2600, 2)
mixw = np.clip((t - (DROP - 1.2)) / 1.2, 0, 1)
pad = cut_dark * (1 - mixw) + cut_open * mixw
# sidechain pump after the drop
pump = np.ones(N)
for k in np.arange(DROP, OUT, BEAT):
    i = int(k * SR)
    n = int(0.42 * SR)
    j = min(N, i + n)
    pump[i:j] = np.minimum(pump[i:j], 0.35 + 0.65 * (np.arange(j - i) / n) ** 0.6)
pad *= pump
# outro: pad swells back on the final chord and fades out
outro_fade = np.clip((DUR - t) / 2.2, 0, 1)
pad_gain = np.where(t < DROP, 0.55 + 0.25 * t / DROP, 0.6)
pad *= pad_gain * outro_fade
add(pad, 0, 0.9, -0.2)
add(np.roll(pad, int(0.011 * SR)), 0, 0.9, 0.2)

# --- final Am chord at the outro (soft bell-ish keys)
for i, m in enumerate([57, 60, 64, 69, 72]):
    n = int(3.9 * SR)
    tt = np.arange(n) / SR
    f = hz(m)
    sig = (np.sin(2 * np.pi * f * tt) + 0.3 * np.sin(2 * np.pi * 2 * f * tt) + 0.08 * np.sin(2 * np.pi * 3.01 * f * tt))
    sig *= env(n, 0.004, 1.4)
    add(sig, OUT + 0.08 + i * 0.09, 0.07, -0.4 + i * 0.2)

# --- kick
def kick():
    n = int(0.35 * SR)
    tt = np.arange(n) / SR
    f = 45 + 85 * np.exp(-tt / 0.03)
    ph = 2 * np.pi * np.cumsum(f) / SR
    return np.sin(ph) * env(n, 0.001, 0.12)


for k in np.arange(DROP, OUT, BEAT):
    add(kick(), k, 0.55)
add(kick(), OUT, 0.5)

# --- hats: soft offbeats; ticking 16ths during the hook
def hat(d=0.03):
    n = int(0.08 * SR)
    return hp(rng.standard_normal(n), 7000) * env(n, 0.001, d)


for k in np.arange(0.0, DROP, BEAT / 2):
    add(hat(0.015), k, 0.035 + 0.03 * k / DROP, 0.3)
for k in np.arange(DROP + BEAT / 2, OUT, BEAT):
    add(hat(0.035), k, 0.07, 0.25)

# --- clap on 2 and 4, very soft
def clap():
    n = int(0.25 * SR)
    x = bp(rng.standard_normal(n), 900, 3000) * env(n, 0.002, 0.08)
    return x


for k in np.arange(DROP + BEAT, OUT, 2 * BEAT):
    add(clap(), k, 0.09, -0.1)

# --- sub bass: root 8ths after the drop
bass = np.zeros(N)
for k in np.arange(DROP, OUT, BEAT / 2):
    bar = int(k // 2.0) % 4
    n = int(0.22 * SR)
    tt = np.arange(n) / SR
    f = hz(ROOTS[bar] - 12 + (12 if (round(k / (BEAT / 2)) % 2) else 0))
    s = np.sin(2 * np.pi * f * tt) + 0.25 * np.sin(2 * np.pi * 2 * f * tt)
    s *= np.clip(tt / 0.005, 0, 1) * np.exp(-tt / 0.16)
    place(bass, s, k)
bass = lp(bass, 400) * pump
add(bass, 0, 0.32)

# --- riser into the drop
n = int(1.3 * SR)
tt = np.arange(n) / SR
noise = rng.standard_normal(n)
riser = np.zeros(n)
for a in range(0, n, 2048):
    b = min(n, a + 2048)
    fc = 400 + 5000 * (a / n) ** 2
    riser[a:b] = bp(noise[a:b] if a == 0 else noise[max(0, a - 512):b], fc, fc * 1.6)[-(b - a):]
riser *= (tt / 1.3) ** 2
add(riser, DROP - 1.3, 0.07)

# --- whooshes on scene changes (filtered noise swell + fall)
def whoosh(len_s=0.45):
    n = int(len_s * SR)
    x = lp(hp(rng.standard_normal(n), 300), 2500)
    tt = np.arange(n) / SR
    e = np.sin(np.pi * np.clip(tt / len_s, 0, 1)) ** 2
    return x * e


# one per hard cut in index.html, landing just before it
for i, at in enumerate([DROP, 7.2, 11.8, 14.8, 19.6, 22.6, 26.6]):
    add(whoosh(), at - 0.3, 0.05, -0.3 if i % 2 else 0.3)
add(whoosh(0.5), OUT - 0.35, 0.07, 0)

# --- plucks on caption reveals, in key (A minor pentatonic)
def pluck(m, d=0.35):
    n = int(0.9 * SR)
    tt = np.arange(n) / SR
    f = hz(m)
    s = np.sin(2 * np.pi * f * tt) * 0.8 + 0.2 * np.sin(2 * np.pi * 2 * f * tt)
    return lp(s * env(n, 0.002, d), 3500)


for at, notes in [(4.75, [76, 81]), (7.35, [72, 76]), (11.95, [74, 79]), (14.95, [76, 81]), (17.6, [81, 84]), (19.75, [72, 76]), (22.75, [74, 79]), (26.75, [76, 81])]:
    for i, m in enumerate(notes):
        add(pluck(m), at + i * 0.125, 0.06, -0.25 + 0.5 * i)

# --- keystrokes: tiny ticks tuned bright but quiet
def tick():
    n = int(0.03 * SR)
    return bp(rng.standard_normal(n), 2500, 6000) * env(n, 0.0005, 0.006)


typing = []
typing += list(np.arange(3.3, 4.1, 0.055))
typing += [7.45 + i / 6 for i in range(6)]
typing += list(np.arange(15.1, 16.5, 0.07))
typing += list(np.arange(22.9, 23.9, 0.06))
for i, k in enumerate(typing):
    add(tick(), k + rng.uniform(-0.008, 0.008), 0.05 * rng.uniform(0.7, 1.0), rng.uniform(-0.3, 0.3))
# enter key on the search, run on the SQL
add(tick(), 8.95, 0.08)
add(tick(), 16.5, 0.08)
add(tick(), 23.9, 0.08)

# --- master
mix = np.stack([L, R], 1)
mix = hp(mix.T, 30).T
fade_in = np.clip(t / 0.03, 0, 1)[:, None]
mix *= fade_in
mix = np.tanh(mix * 1.4) / np.tanh(1.4)
mix *= 0.89 / np.max(np.abs(mix))
wavfile.write(sys.argv[1], SR, (mix * 32767).astype(np.int16))
