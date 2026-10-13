"""Build the Ciyuan logo assets from the supplied SVG. Development only."""
import json
import re
import xml.etree.ElementTree as ET
from pathlib import Path
import build_art as build
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
SVG = ROOT / 'pangu/assets/ciyuan-mark.svg'
ns = {'s': 'http://www.w3.org/2000/svg'}
tree = ET.parse(SVG)
shapes = []
for path in tree.findall('.//s:path', ns):
    gid = path.get('fill')[5:-1]
    grad = tree.find('.//s:linearGradient[@id="%s"]' % gid, ns)
    stops = []
    for stop in grad:
        h = stop.get('stop-color').lstrip('#')
        stops.append([float(stop.get('offset', '0')), [int(h[i:i+2], 16) for i in (0, 2, 4)]])
    for poly in build.parse_path(path.get('d')):
        shapes.append({'points': poly, 'axis': [float(grad.get(k)) for k in ('x1', 'y1', 'x2', 'y2')], 'stops': stops})
asset = ROOT / 'pangu/assets/art.json'
data = json.loads(asset.read_text())
data.update(source='ciyuan-mark.svg — user supplied logo', petals=[s['points'] for s in shapes],
            shapes=shapes, logo_bbox=[0, 0, 71, 49], origin=[35.5, 24.5], word=[],
            word_bbox=[0, 0, 4, 1], gradient={'cx': 0, 'cy': 0, 'r': 71,
            'stops': [[0, [61, 206, 242]], [.5, [0, 109, 255]], [1, [8, 63, 186]]]})
build.FONT = build.default_font()
assert build.FONT, 'PingFang font is required for building assets'
def english_word(rows):
    text = 'TOKENS INFINITY'
    font = ImageFont.truetype('/System/Library/Fonts/Supplemental/Arial Bold.ttf', 160)
    box = font.getbbox(text)
    img = Image.new('L', (box[2]-box[0], box[3]-box[1]), 0)
    ImageDraw.Draw(img).text((-box[0], -box[1]), text, font=font, fill=255)
    height = rows * 4
    width = round(img.width / img.height * height * build.SUB_ASPECT)
    width += width % 2
    img = img.resize((width, height), Image.Resampling.LANCZOS)
    bits = [format(sum(1 << x for x in range(width) if img.getpixel((x,y)) >= 118), 'x')
            for y in range(height)]
    return {'rows': rows, 'cols': width//2, 'bits': bits}


data['word_text'] = 'TOKENS INFINITY'
data['word_sizes'] = {str(r): english_word(r) for r in range(1, 7)}
data['slogans'] = {k: {'text': '从词元，抵达无限', 'sizes': [build.text_bits('从词元，抵达无限', r) for r in (3, 4, 5, 6)]} for k in ('codex', 'claude')}
asset.write_text(json.dumps(data, ensure_ascii=False, separators=(',', ':')))
print('Built six logo faces with original SVG gradients and TOKENS INFINITY wordmark')
