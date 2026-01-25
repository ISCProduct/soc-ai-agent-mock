#!/usr/bin/env python3
import argparse
import json
import os

import fitz  # PyMuPDF
from PIL import Image, ImageDraw, ImageFont
import io


def resolve_japanese_font():
    env_path = os.getenv("ANNOTATION_FONT_PATH", "").strip()
    if env_path and os.path.exists(env_path):
        return env_path
    candidates = [
        "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
        "/usr/share/fonts/opentype/noto/NotoSansCJKjp-Regular.otf",
        "/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
        "/usr/share/fonts/truetype/noto/NotoSansCJKjp-Regular.otf",
        "/usr/share/fonts/opentype/noto/NotoSansJP-Regular.otf",
        "/usr/share/fonts/truetype/noto/NotoSansJP-Regular.otf",
        "/usr/share/fonts/truetype/fonts-japanese-gothic.ttf",
        "/System/Library/Fonts/AppleGothic.ttf",
        "/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc",
        "/System/Library/Fonts/ヒラギノ角ゴシック W6.ttc",
        "/System/Library/Fonts/ヒラギノ明朝 ProN W3.otf",
        "/Library/Fonts/Arial Unicode.ttf",
    ]
    for path in candidates:
        if os.path.exists(path):
            return path
    return None


def resolve_note_rect(page_rect, target_rect):
    margin = 6
    note_height = 60
    note_width = 220

    center_y = (target_rect.y0 + target_rect.y1) / 2
    top = max(page_rect.y0 + margin, center_y - note_height / 2)
    bottom = min(page_rect.y1 - margin, top + note_height)
    top = bottom - note_height

    right_space = page_rect.x1 - target_rect.x1 - margin
    left_space = target_rect.x0 - page_rect.x0 - margin

    if right_space >= note_width:
        left = target_rect.x1 + margin
        right = left + note_width
    elif left_space >= note_width:
        right = target_rect.x0 - margin
        left = right - note_width
    else:
        left = max(page_rect.x0 + margin, target_rect.x0)
        right = min(page_rect.x1 - margin, left + note_width)
        left = right - note_width

    note_rect = fitz.Rect(left, top, right, bottom)
    if note_rect.y0 < page_rect.y0 + margin:
        note_rect.y1 = note_rect.y1 + (page_rect.y0 + margin - note_rect.y0)
        note_rect.y0 = page_rect.y0 + margin
    if note_rect.y1 > page_rect.y1 - margin:
        note_rect.y0 = note_rect.y0 - (note_rect.y1 - (page_rect.y1 - margin))
        note_rect.y1 = page_rect.y1 - margin

    return note_rect


def draw_callout(page, target_rect, note_rect, border_color):
    target_x = (target_rect.x0 + target_rect.x1) / 2
    if note_rect.y1 <= target_rect.y0:
        base_y = note_rect.y1
        target_y = target_rect.y0
    else:
        base_y = note_rect.y0
        target_y = target_rect.y1

    base_x = min(max(target_x, note_rect.x0 + 10), note_rect.x1 - 10)
    page.draw_line((base_x, base_y), (target_x, target_y), color=border_color, width=1)


def wrap_text(draw, text, font, max_width):
    lines = []
    for raw_line in text.split("\n"):
        line = raw_line.strip()
        if not line:
            lines.append("")
            continue
        if " " in line:
            words = line.split()
        else:
            words = list(line)
        current = ""
        for word in words:
            test = word if current == "" else f"{current} {word}" if " " in line else current + word
            bbox = draw.textbbox((0, 0), test, font=font)
            if bbox[2] <= max_width:
                current = test
            else:
                if current:
                    lines.append(current)
                current = word
        if current:
            lines.append(current)
    return lines


def render_note_image(text, width_pt, height_pt, font_path):
    scale = 2
    width_px = max(1, int(width_pt * scale))
    height_px = max(1, int(height_pt * scale))
    margin = 6 * scale
    font_size = 8 * scale
    if font_path.lower().endswith(".ttc"):
        font = ImageFont.truetype(font_path, font_size, index=0)
    else:
        font = ImageFont.truetype(font_path, font_size)

    image = Image.new("RGB", (width_px, height_px), (255, 250, 230))
    draw = ImageDraw.Draw(image)
    draw.rectangle([0, 0, width_px - 1, height_px - 1], outline=(150, 110, 40), width=2)

    max_width = width_px - margin * 2
    lines = wrap_text(draw, text, font, max_width)
    y = margin
    for line in lines:
        if y + font_size > height_px - margin:
            break
        draw.text((margin, y), line, font=font, fill=(40, 30, 20))
        y += int(font_size * 1.2)

    buffer = io.BytesIO()
    image.save(buffer, format="PNG")
    return buffer.getvalue()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--items", required=True)
    args = parser.parse_args()

    if not os.path.exists(args.input):
        raise SystemExit(f"input file not found: {args.input}")
    if not os.path.exists(args.items):
        raise SystemExit(f"items file not found: {args.items}")

    with open(args.items, "r", encoding="utf-8") as f:
        items = json.load(f)

    doc = fitz.open(args.input)
    font_path = resolve_japanese_font()
    if not font_path:
        raise SystemExit("Japanese font not found. Set ANNOTATION_FONT_PATH.")
    for item in items:
        page_number = item.get("page_number", 1)
        bbox = item.get("bbox", [20, 20, 200, 60])
        page_width = item.get("page_width")
        page_height = item.get("page_height")
        message = item.get("message", "")
        suggestion = item.get("suggestion", "")

        page_index = max(0, page_number - 1)
        if page_index >= len(doc):
            continue

        page = doc[page_index]
        rect = fitz.Rect(bbox[0], bbox[1], bbox[2], bbox[3])
        page_rect = page.rect

        if page_width and page_height:
            scale_x = page_rect.width / float(page_width)
            scale_y = page_rect.height / float(page_height)
            rect = fitz.Rect(
                rect.x0 * scale_x,
                rect.y0 * scale_y,
                rect.x1 * scale_x,
                rect.y1 * scale_y,
            )
        else:
            if 0 <= rect.x0 <= 1.5 and 0 <= rect.y0 <= 1.5 and 0 <= rect.x1 <= 1.5 and 0 <= rect.y1 <= 1.5:
                rect = fitz.Rect(
                    rect.x0 * page_rect.width,
                    rect.y0 * page_rect.height,
                    rect.x1 * page_rect.width,
                    rect.y1 * page_rect.height,
                )
        page.draw_rect(rect, color=(1, 0.2, 0.2), width=1.5)
        page.draw_rect(rect, color=(1, 0.9, 0.6), fill=(1, 0.9, 0.6), width=0)

        note = message
        if suggestion:
            note = f"{message}\n改善案: {suggestion}"
        if note:
            note_rect = resolve_note_rect(page_rect, rect)
            draw_callout(page, rect, note_rect, (0.6, 0.4, 0.1))
            note_image = render_note_image(note, note_rect.width, note_rect.height, font_path)
            page.insert_image(note_rect, stream=note_image, keep_proportion=False, overlay=True)

    doc.save(args.output, garbage=4, deflate=True)
    doc.close()


if __name__ == "__main__":
    main()
