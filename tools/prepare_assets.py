from PIL import Image, ImageDraw, ImageFilter
from collections import deque
import os

root = r'H:\CodeFromGithub\All-in-One-Mouse-Driver'
src = os.path.join(root, '鼠标图片')
out = os.path.join(root, 'assets')
os.makedirs(os.path.join(out, 'mice'), exist_ok=True)


def cut_pure_white_bg(im):
    """Flood-fill exact pure white (255,255,255) from the border only.

    Global replace leaves a light anti-aliased halo on the silhouette; flood
    from the edge removes the studio white without punching holes in the body.
    """
    im = im.convert('RGBA')
    w, h = im.size
    px = im.load()
    q = deque()
    seen = bytearray(w * h)

    def pure_white(x, y):
        r, g, b, a = px[x, y]
        return a > 0 and r == 255 and g == 255 and b == 255

    for x in range(w):
        for y in (0, h - 1):
            i = y * w + x
            if not seen[i] and pure_white(x, y):
                seen[i] = 1
                q.append((x, y))
    for y in range(h):
        for x in (0, w - 1):
            i = y * w + x
            if not seen[i] and pure_white(x, y):
                seen[i] = 1
                q.append((x, y))
    while q:
        x, y = q.popleft()
        r, g, b, a = px[x, y]
        px[x, y] = (r, g, b, 0)
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            nx, ny = x + dx, y + dy
            if 0 <= nx < w and 0 <= ny < h:
                ni = ny * w + nx
                if not seen[ni] and pure_white(nx, ny):
                    seen[ni] = 1
                    q.append((nx, ny))

    # Erode the anti-aliased silhouette: pull a few light pixels into the
    # background so the dark UI does not show a bright fringe.
    for _ in range(3):
        to_clear = []
        for y in range(h):
            for x in range(w):
                r, g, b, a = px[x, y]
                if a == 0:
                    continue
                # light enough to be part of the white studio edge
                if min(r, g, b) < 238:
                    continue
                for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    nx, ny = x + dx, y + dy
                    if 0 <= nx < w and 0 <= ny < h and seen[ny * w + nx]:
                        to_clear.append((x, y))
                        break
        if not to_clear:
            break
        for x, y in to_clear:
            r, g, b, a = px[x, y]
            px[x, y] = (r, g, b, 0)
            seen[y * w + x] = 1
    bbox = im.getbbox()
    if bbox:
        im = im.crop(bbox)
    # Soften the remaining hard cut so it does not look pixelated.
    a = im.getchannel('A').filter(ImageFilter.GaussianBlur(0.45))
    im.putalpha(a)
    return im


def fit_premultiplied(im, max_w=520, max_h=520):
    """Resize in premultiplied space so transparent white does not bleed back."""
    im = im.convert('RGBA')
    w, h = im.size
    px = im.load()
    for y in range(h):
        for x in range(w):
            r, g, b, a = px[x, y]
            if a == 255:
                continue
            if a == 0:
                px[x, y] = (0, 0, 0, 0)
                continue
            px[x, y] = (r * a // 255, g * a // 255, b * a // 255, a)
    im = im.copy()
    im.thumbnail((max_w, max_h), Image.Resampling.LANCZOS)
    w, h = im.size
    px = im.load()
    for y in range(h):
        for x in range(w):
            r, g, b, a = px[x, y]
            if a == 0:
                px[x, y] = (0, 0, 0, 0)
            elif a < 255:
                if a == 0:
                    continue
                r = min(255, r * 255 // a)
                g = min(255, g * 255 // a)
                b = min(255, b * 255 // a)
                # residual pure white from blend -> drop
                if r >= 254 and g >= 254 and b >= 254 and a < 200:
                    px[x, y] = (0, 0, 0, 0)
                else:
                    px[x, y] = (r, g, b, a)
    bbox = im.getbbox()
    if bbox:
        im = im.crop(bbox)
    return im


inca = Image.open(os.path.join(src, 'INCA-2.png'))
inca = cut_pure_white_bg(inca)
inca = fit_premultiplied(inca)
inca.save(os.path.join(out, 'mice', 'lamzu-inca.png'), optimize=True)
print('inca', inca.size)

for src_name, dst_name in [
    ('vaxee_e1_wireless_blue_1.png', 'vaxee-e1.png'),
    ('vaxee_zygen_np-01s_v2_wireless_blue_1.png', 'vaxee-np01sv2.png'),
]:
    im = Image.open(os.path.join(src, src_name)).convert('RGBA')
    bbox = im.getbbox()
    if bbox:
        im = im.crop(bbox)
    im = im.copy()
    im.thumbnail((520, 520), Image.Resampling.LANCZOS)
    im.save(os.path.join(out, 'mice', dst_name), optimize=True)
    print(dst_name, im.size)


def make_logo(size=256):
    s = size
    img = Image.new('RGBA', (s, s), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    pad = int(s * 0.06)
    d.rounded_rectangle([pad, pad, s - pad, s - pad], radius=int(s * 0.22), fill=(22, 28, 42, 255))
    d.rounded_rectangle(
        [pad, pad, s - pad, s - pad],
        radius=int(s * 0.22),
        outline=(155, 185, 255, 255),
        width=max(2, s // 64),
    )
    mx, my = s * 0.5, s * 0.50
    bw, bh = s * 0.30, s * 0.40
    d.ellipse([mx - bw / 2, my - bh / 2, mx + bw / 2, my + bh / 2], fill=(210, 220, 240, 255))
    d.line([(mx, my - bh / 2 + s * 0.05), (mx, my - s * 0.02)], fill=(90, 100, 120, 255), width=max(1, s // 80))
    d.rounded_rectangle(
        [mx - s * 0.035, my - bh / 2 + s * 0.10, mx + s * 0.035, my - bh / 2 + s * 0.22],
        radius=s * 0.03,
        fill=(40, 48, 60, 255),
    )
    for i, (dx, dy) in enumerate([(-0.30, 0.32), (0.0, 0.36), (0.30, 0.32)]):
        r = s * 0.045
        cx, cy = mx + s * dx, my + s * dy
        color = (155, 185, 255, 255) if i == 1 else (120, 140, 180, 255)
        d.ellipse([cx - r, cy - r, cx + r, cy + r], fill=color)
    return img


logo = make_logo(256)
logo.save(os.path.join(out, 'logo-256.png'), optimize=True)
logo.save(
    os.path.join(out, 'app.ico'),
    format='ICO',
    sizes=[(16, 16), (32, 32), (48, 48), (256, 256)],
)
print('done')
