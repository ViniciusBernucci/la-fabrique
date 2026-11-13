"""Reproduce the supplied 10s/24fps clip as a silent cyclic visual; no app dependencies."""
import os
import subprocess
import sys
from pathlib import Path

ffmpeg = os.environ.get('FFMPEG_BINARY', 'ffmpeg')
source = sys.argv[1]
output = Path(sys.argv[2])
poster = Path(sys.argv[3])
output.parent.mkdir(parents=True, exist_ok=True)
poster.parent.mkdir(parents=True, exist_ok=True)
# Begin at 0.75s; blend 18 tail/head frames with both fade endpoints included.
# Duration = 17/24s between their timestamps, avoiding a residual tail at wrap.
# The crossfade lives inside the encoded clip; native looping joins adjacent frames.
filters = ('[0:v]scale=960:540,split=2[body][head];'
           '[body]trim=start=0.75:end=10,setpts=PTS-STARTPTS,fps=24,settb=AVTB[b];'
           '[head]trim=start=0:end=0.75,setpts=PTS-STARTPTS,fps=24,settb=AVTB[h];'
           '[b][h]xfade=transition=fade:duration=0.708333:offset=8.5,format=yuv420p[out]')
subprocess.run([ffmpeg, '-hide_banner', '-loglevel', 'error', '-y', '-i', source,
                '-filter_complex', filters, '-map', '[out]', '-an', '-c:v', 'libx264',
                '-preset', 'medium', '-crf', '23', '-g', '24', '-movflags', '+faststart',
                '-map_metadata', '-1', str(output)], check=True)
subprocess.run([ffmpeg, '-hide_banner', '-loglevel', 'error', '-y', '-i', str(output),
                '-frames:v', '1', '-q:v', '3', str(poster)], check=True)
