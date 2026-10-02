#!/usr/bin/env python3
"""Render fictional demo data as theme-colored SVG. Development tool only."""
import html
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
PALETTES = {
    "dashboard": {"background":"#181425", "foreground":"#E0DAF0", "accent":"#8BD5CA", "color5":"#C6A0F6", "color2":"#A6DA95", "color8":"#9A93AC", "color1":"#ED8796"},
    "dashboard-light": {"background":"#F5F1EB", "foreground":"#292735", "accent":"#006D77", "color5":"#7552A3", "color2":"#376A32", "color8":"#6D6676", "color1":"#A73144"},
}
SGR = re.compile(r"\x1b\[([0-9;]*)m")
for name, palette in PALETTES.items():
    with tempfile.TemporaryDirectory(prefix="tempo-art-") as tmp:
        home = Path(tmp)
        theme = home / ".local/state/omarchy/current/theme"
        theme.mkdir(parents=True)
        (theme / "colors.toml").write_text("\n".join(f'{k} = "{v}"' for k, v in palette.items()))
        rendered = subprocess.check_output([str(ROOT / "bin/tempo"), "--render"], env={**os.environ, "HOME":tmp}, text=True)
    lines = rendered.splitlines()
    height = len(lines)*19 + 60
    svg = [f'<svg xmlns="http://www.w3.org/2000/svg" width="1120" height="{height}" viewBox="0 0 1120 {height}">',
           f'<rect width="100%" height="100%" rx="14" fill="{palette["background"]}"/>',
           '<g font-family="DejaVu Sans Mono,monospace" font-size="14" xml:space="preserve">']
    for y, line in enumerate(lines):
        fill, bold, previous = palette["foreground"], False, 0
        spans=[]
        for match in SGR.finditer(line):
            part=line[previous:match.start()]
            if part:
                spans.append(f'<tspan fill="{fill}" font-weight="{"bold" if bold else "normal"}">{html.escape(part).replace(" ", "&#160;")}</tspan>')
            codes=[int(v or 0) for v in match.group(1).split(";")]
            i=0
            while i<len(codes):
                c=codes[i]
                if c==0: fill,bold=palette["foreground"],False
                elif c==1: bold=True
                elif c==22: bold=False
                elif c==39: fill=palette["foreground"]
                elif c==38 and i+4<len(codes) and codes[i+1]==2:
                    fill="#"+"".join(f'{v:02x}' for v in codes[i+2:i+5]);i+=4
                i+=1
            previous=match.end()
        if line[previous:]: spans.append(f'<tspan fill="{fill}">{html.escape(line[previous:]).replace(" ", "&#160;")}</tspan>')
        svg.append(f'<text x="24" y="{35+y*19}">'+"".join(spans)+'</text>')
    svg+=['</g></svg>']
    (ROOT / "assets" / f"{name}.svg").write_text("\n".join(svg)+"\n")
