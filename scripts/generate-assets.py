#!/usr/bin/env python3
"""Generate every image the project ships: the application icon and the
Inno Setup wizard graphics.

The artwork is an upward arrow — "a newer version is available" — in GitHub's
dark/green palette, drawn on a rounded square.

Everything is rendered at 4x and downsampled, which is cheaper than fighting
Pillow's lack of antialiased polygon drawing.

Run from the repository root:

    python scripts/generate-assets.py

Outputs (all committed, so building never requires Python):
    assets/icon.ico                     multi-resolution app icon
    assets/icon.png                     256px, for docs
    installer/windows/wizard-large*.png left-hand image in the installer
    installer/windows/wizard-small*.png corner image on installer pages
"""

from pathlib import Path

from PIL import Image, ImageDraw

# GitHub-ish palette.
BG_TOP = (45, 51, 59)  # #2D333B
BG_BOTTOM = (22, 27, 34)  # #161B22
ACCENT = (63, 185, 80)  # #3FB950

SS = 4  # supersampling factor

ROOT = Path(__file__).resolve().parent.parent
ASSETS = ROOT / "assets"
INSTALLER = ROOT / "installer" / "windows"


def vertical_gradient(size, top, bottom):
    """A simple top-to-bottom gradient."""
    width, height = size
    img = Image.new("RGB", (1, height))
    for y in range(height):
        t = y / max(height - 1, 1)
        img.putpixel(
            (0, y),
            (
                round(top[0] + (bottom[0] - top[0]) * t),
                round(top[1] + (bottom[1] - top[1]) * t),
                round(top[2] + (bottom[2] - top[2]) * t),
            ),
        )
    return img.resize((width, height), Image.NEAREST)


def arrow_polygon(cx, cy, width, height):
    """Points of an upward arrow centred on (cx, cy).

    The head is the top 55% and spans the full width; the shaft is 38% as wide.
    """
    half_w = width / 2
    half_h = height / 2
    head_bottom = cy - half_h + height * 0.55
    shaft_half = width * 0.19

    return [
        (cx, cy - half_h),  # tip
        (cx + half_w, head_bottom),  # right barb
        (cx + shaft_half, head_bottom),  # right shoulder
        (cx + shaft_half, cy + half_h),  # right foot
        (cx - shaft_half, cy + half_h),  # left foot
        (cx - shaft_half, head_bottom),  # left shoulder
        (cx - half_w, head_bottom),  # left barb
    ]


def draw_icon(size, padding_ratio=0.0, radius_ratio=0.22):
    """Rounded square with the arrow on it, returned as RGBA at `size` px."""
    s = size * SS
    pad = round(s * padding_ratio)
    box = s - 2 * pad

    canvas = Image.new("RGBA", (s, s), (0, 0, 0, 0))

    # Rounded-square background, filled with the gradient through a mask.
    mask = Image.new("L", (box, box), 0)
    ImageDraw.Draw(mask).rounded_rectangle(
        [0, 0, box - 1, box - 1], radius=round(box * radius_ratio), fill=255
    )
    plate = vertical_gradient((box, box), BG_TOP, BG_BOTTOM).convert("RGBA")
    canvas.paste(plate, (pad, pad), mask)

    # The arrow, slightly above centre so it looks optically balanced.
    draw = ImageDraw.Draw(canvas)
    cx = s / 2
    cy = s / 2 - box * 0.01
    draw.polygon(
        arrow_polygon(cx, cy, box * 0.52, box * 0.58),
        fill=ACCENT,
    )

    return canvas.resize((size, size), Image.LANCZOS)


def draw_wizard_large(size):
    """The tall image down the left-hand side of the installer."""
    width, height = size
    w, h = width * SS, height * SS

    img = vertical_gradient((w, h), BG_TOP, BG_BOTTOM).convert("RGBA")
    draw = ImageDraw.Draw(img)

    # Oversized arrow in the upper third.
    cx = w / 2
    cy = h * 0.34
    draw.polygon(arrow_polygon(cx, cy, w * 0.54, h * 0.32), fill=ACCENT)

    # Three ticks below, suggesting a checked list of actions.
    bar_w = w * 0.42
    bar_h = max(h * 0.011, SS)
    for i, alpha in enumerate((150, 100, 60)):
        y = h * 0.68 + i * h * 0.052
        overlay = Image.new("RGBA", img.size, (0, 0, 0, 0))
        ImageDraw.Draw(overlay).rounded_rectangle(
            [cx - bar_w / 2, y, cx - bar_w / 2 + bar_w * (1 - i * 0.22), y + bar_h],
            radius=bar_h / 2,
            fill=(255, 255, 255, alpha),
        )
        img = Image.alpha_composite(img, overlay)

    return img.resize((width, height), Image.LANCZOS)


def main():
    ASSETS.mkdir(parents=True, exist_ok=True)
    INSTALLER.mkdir(parents=True, exist_ok=True)

    # Application icon. 256 first so the .ico carries a crisp large frame.
    icon_sizes = [256, 128, 64, 48, 32, 24, 16]
    frames = [draw_icon(s) for s in icon_sizes]
    frames[0].save(
        ASSETS / "icon.ico",
        format="ICO",
        sizes=[(s, s) for s in icon_sizes],
    )
    frames[0].save(ASSETS / "icon.png", format="PNG")
    print(f"wrote {ASSETS / 'icon.ico'} ({', '.join(str(s) for s in icon_sizes)})")
    print(f"wrote {ASSETS / 'icon.png'} (256)")

    # Wizard images. Inno picks the closest match per DPI, so ship two sizes.
    # Large keeps the documented 164:314 aspect ratio.
    for w, h in ((240, 459), (480, 918)):
        path = INSTALLER / f"wizard-large-{w}x{h}.png"
        draw_wizard_large((w, h)).convert("RGB").save(path, format="PNG")
        print(f"wrote {path}")

    for s in (147, 294):
        path = INSTALLER / f"wizard-small-{s}x{s}.png"
        # Padded so the plate does not touch the edges of the wizard's corner.
        draw_icon(s, padding_ratio=0.06).save(path, format="PNG")
        print(f"wrote {path}")


if __name__ == "__main__":
    main()
