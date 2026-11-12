"""Compare the cyclic boundary with ordinary adjacent frames (decoded RGB thumbnail)."""
import hashlib
import json
import os
import statistics
import subprocess
import sys
from pathlib import Path
source, asset, output = map(Path, sys.argv[1:4])
raw = subprocess.check_output([os.environ.get('FFMPEG_BINARY', 'ffmpeg'), '-hide_banner',
                               '-loglevel', 'error', '-i', str(asset), '-vf', 'scale=64:36',
                               '-f', 'rawvideo', '-pix_fmt', 'rgb24', '-'])
size = 64 * 36 * 3
frames = [raw[i:i+size] for i in range(0, len(raw), size)]
def difference(a, b):
    return sum(abs(x-y) for x, y in zip(a, b)) / len(a)
adjacent = [difference(a, b) for a, b in zip(frames, frames[1:])]
seam = difference(frames[-1], frames[0])
percentile = sorted(adjacent)[int(len(adjacent)*.95)]
result = dict(sourceSha256=hashlib.sha256(source.read_bytes()).hexdigest(),
              assetSha256=hashlib.sha256(asset.read_bytes()).hexdigest(),
              sourceBytes=source.stat().st_size, assetBytes=asset.stat().st_size,
              frames=len(frames), seamMeanAbsoluteDifference=seam,
              adjacentFrameMedian=statistics.median(adjacent),
              adjacentFrame95thPercentile=percentile, seamBelow95thPercentile=seam<percentile)
output.write_text(json.dumps(result, indent=2)+'\n')
print(json.dumps(result))
assert seam < percentile, result
