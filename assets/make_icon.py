"""Генератор иконки caps-switch: клавиша с A/Я и двусторонней стрелкой."""
import os
from PIL import Image, ImageDraw, ImageFont, ImageFilter

HERE = os.path.dirname(os.path.abspath(__file__))
OUT_PNG = os.path.join(HERE, "icon.png")
OUT_ICO = os.path.join(HERE, "icon.ico")

S = 1024
c1 = (79, 70, 229)     # indigo
c2 = (124, 58, 237)    # violet
ink = (30, 27, 75)     # тёмно-индиго для букв
accent = (124, 58, 237)

# --- фон-градиент ---
bg = Image.new("RGB", (S, S), c1)
bd = ImageDraw.Draw(bg)
for y in range(S):
    t = y / (S - 1)
    r = round(c1[0] + (c2[0] - c1[0]) * t)
    g = round(c1[1] + (c2[1] - c1[1]) * t)
    b = round(c1[2] + (c2[2] - c1[2]) * t)
    bd.line([(0, y), (S, y)], fill=(r, g, b))

img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
mask = Image.new("L", (S, S), 0)
ImageDraw.Draw(mask).rounded_rectangle(
    [0, 0, S - 1, S - 1], radius=int(S * 0.22), fill=255
)
img.paste(bg.convert("RGBA"), (0, 0), mask)

draw = ImageDraw.Draw(img)

# --- тень клавиши ---
sh = Image.new("RGBA", (S, S), (0, 0, 0, 0))
ImageDraw.Draw(sh).rounded_rectangle(
    [int(S * 0.18), int(S * 0.24), int(S * 0.82), int(S * 0.88)],
    radius=int(S * 0.12), fill=(8, 8, 28, 120),
)
sh = sh.filter(ImageFilter.GaussianBlur(int(S * 0.035)))
img.alpha_composite(sh)
draw = ImageDraw.Draw(img)

# --- клавиша ---
kx0, ky0 = int(S * 0.17), int(S * 0.18)
kx1, ky1 = int(S * 0.83), int(S * 0.82)
rad = int(S * 0.12)
draw.rounded_rectangle([kx0, ky0, kx1, ky1], radius=rad, fill=(248, 250, 252, 255))
# нижняя грань для объёма
draw.rounded_rectangle(
    [kx0, ky1 - int(S * 0.05), kx1, ky1], radius=rad, fill=(226, 232, 240, 255)
)
draw.rounded_rectangle([kx0, ky0, kx1, ky1], radius=rad, outline=(203, 213, 225, 255), width=int(S * 0.008))

# --- шрифт ---
font = None
for name in ("seguisb.ttf", "segoeuib.ttf", "arialbd.ttf", "segoeui.ttf"):
    p = os.path.join("C:\\Windows\\Fonts", name)
    if os.path.exists(p):
        font = ImageFont.truetype(p, int(S * 0.33))
        break
if font is None:
    font = ImageFont.load_default()

cy = int(S * 0.47)
draw.text((int(S * 0.33), cy), "A", font=font, fill=ink + (255,), anchor="mm")
draw.text((int(S * 0.67), cy), "Я", font=font, fill=ink + (255,), anchor="mm")

# --- двусторонняя стрелка в просвете ---
cx = int(S * 0.5)
aw = int(S * 0.085)
lw = int(S * 0.017)
head = int(S * 0.028)
y1 = cy - int(S * 0.045)
y2 = cy + int(S * 0.045)
# верхняя -> вправо
draw.line([(cx - aw // 2, y1), (cx + aw // 2, y1)], fill=accent + (255,), width=lw)
draw.polygon(
    [(cx + aw // 2, y1),
     (cx + aw // 2 - head, y1 - head),
     (cx + aw // 2 - head, y1 + head)],
    fill=accent + (255,),
)
# нижняя -> влево
draw.line([(cx + aw // 2, y2), (cx - aw // 2, y2)], fill=accent + (255,), width=lw)
draw.polygon(
    [(cx - aw // 2, y2),
     (cx - aw // 2 + head, y2 - head),
     (cx - aw // 2 + head, y2 + head)],
    fill=accent + (255,),
)

img.resize((512, 512), Image.LANCZOS).save(OUT_PNG)
img.save(OUT_ICO, sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
print("ok:", OUT_PNG, OUT_ICO)
