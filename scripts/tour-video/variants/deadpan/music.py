import sys
import numpy as np
from scipy.signal import butter, sosfilt
from scipy.io import wavfile

# Deadpan: a slow felt piano over a quiet pad, no drums. The only effects are a soft
# wood tock on each scene change and keystrokes, all in A minor and kept far back.
SR = 44100
DUR = 20.0
N = int(SR * DUR)
t = np.arange(N) / SR
rng = np.random.default_rng(3)
BEAT = 60 / 72


def hz(m):
    return 440.0 * 2 ** ((m - 69) / 12)


def lp(x, f, o=2):
    return sosfilt(butter(o, f, "low", fs=SR, output="sos"), x)


def bp(x, lo, hi, o=2):
    return sosfilt(butter(o, [lo, hi], "band", fs=SR, output="sos"), x)


L = np.zeros(N)
R = np.zeros(N)


def add(sig, at, gain=1.0, pan=0.0):
    i = int(at * SR)
    j = min(N, i + len(sig))
    if i >= N:
        return
    L[i:j] += sig[: j - i] * gain * np.sqrt(0.5 * (1 - pan))
    R[i:j] += sig[: j - i] * gain * np.sqrt(0.5 * (1 + pan))


def piano(m, length=3.0):
    n = int(length * SR)
    tt = np.arange(n) / SR
    f = hz(m)
    s = sum(a * np.sin(2 * np.pi * f * k * tt) * np.exp(-tt * (1.2 + k * 0.9)) for k, a in [(1, 1), (2, 0.35), (3, 0.12), (4, 0.05)])
    hammer = lp(rng.standard_normal(n), 1800) * np.exp(-tt / 0.004) * 0.15
    att = np.clip(tt / 0.006, 0, 1)
    return lp((s + hammer) * att, 2400)


# pad: Am, F, C, E over the whole piece, very quiet
CHORDS = [[45, 57, 60, 64], [41, 57, 60, 65], [48, 55, 60, 64], [40, 56, 59, 64]]
pad = np.zeros(N)
for bar in range(5):
    s0 = bar * 4.0
    seg = (t >= s0) & (t < s0 + 4.6)
    tt = t[seg] - s0
    sig = sum(np.sin(2 * np.pi * hz(m) * tt + 0.3 * np.sin(2 * np.pi * 0.2 * tt)) for m in CHORDS[bar % 4])
    sig *= np.clip(tt / 0.8, 0, 1) * np.clip((4.6 - tt) / 0.8, 0, 1)
    pad[seg] += sig
pad = lp(pad, 900) * np.clip((DUR - t) / 2.5, 0, 1)
add(pad, 0, 0.05, -0.3)
add(np.roll(pad, int(0.013 * SR)), 0, 0.05, 0.3)

# piano: one note per beat, a plain falling figure that never resolves until the end
MOTIF = [76, 72, 69, None, 77, 72, 69, None, 76, 72, 67, None, 76, 71, 68, None]
k = 0
for b in np.arange(0.3, 15.6, BEAT):
    m = MOTIF[k % len(MOTIF)]
    k += 1
    if m is not None:
        add(piano(m), b, 0.16, rng.uniform(-0.2, 0.2))
# sign-off: a single A minor chord, rolled slowly
for i, m in enumerate([57, 64, 69, 72]):
    add(piano(m, 4.0), 15.8 + i * 0.07, 0.13, -0.3 + 0.2 * i)


def tock(m=69):
    n = int(0.25 * SR)
    tt = np.arange(n) / SR
    s = np.sin(2 * np.pi * hz(m) * tt) * np.exp(-tt / 0.03) + bp(rng.standard_normal(n), 600, 1800) * np.exp(-tt / 0.008) * 0.3
    return s


for at in (4.4, 10.0, 15.6):
    add(tock(), at, 0.07)


def tick():
    n = int(0.02 * SR)
    tt = np.arange(n) / SR
    return bp(rng.standard_normal(n), 2000, 5000) * np.exp(-tt / 0.005)


for at in np.arange(10.9, 12.7, 0.075):
    add(tick(), at + rng.uniform(-0.01, 0.01), 0.03, rng.uniform(-0.3, 0.3))

mix = np.stack([L, R], 1)
mix *= np.clip(t / 0.02, 0, 1)[:, None]
mix = np.tanh(mix * 1.2) / np.tanh(1.2)
mix *= 0.89 / np.max(np.abs(mix))
wavfile.write(sys.argv[1], SR, (mix * 32767).astype(np.int16))
