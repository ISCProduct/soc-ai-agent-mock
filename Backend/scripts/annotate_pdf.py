#!/usr/bin/env python3
import argparse
import io
import json
import os

import fitz  # PyMuPDF
from PIL import Image, ImageDraw, ImageFont

SEVERITY_COLORS = {
    "critical": {
        "highlight": (1.0, 0.82, 0.82),
        "border": (0.78, 0.15, 0.15),
        "badge": (0.85, 0.15, 0.15),
        "label_fg": (1.0, 1.0, 1.0),
        "ja": "重大",
        "pil_chip": (211, 47, 47),
    },
    "warning": {
        "highlight": (1.0, 0.95, 0.75),
        "border": (0.75, 0.50, 0.05),
        "badge": (0.85, 0.55, 0.05),
        "label_fg": (1.0, 1.0, 1.0),
        "ja": "注意",
        "pil_chip": (237, 108, 2),
    },
    "info": {
        "highlight": (0.82, 0.92, 1.0),
        "border": (0.18, 0.48, 0.78),
        "badge": (0.18, 0.48, 0.78),
        "label_fg": (1.0, 1.0, 1.0),
        "ja": "情報",
        "pil_chip": (2, 136, 209),
    },
}

DEFAULT_COLOR = {
    "highlight": (0.90, 0.90, 0.90),
    "border": (0.40, 0.40, 0.40),
    "badge": (0.40, 0.40, 0.40),
    "label_fg": (1.0, 1.0, 1.0),
    "ja": "",
    "pil_chip": (120, 120, 120),
}


def get_color(severity):
    return SEVERITY_COLORS.get(severity, DEFAULT_COLOR)


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


def wrap_text(draw, text, font, max_width):
    lines = []
    for raw_line in text.split("\n"):
        line = raw_line.strip()
        if not line:
            lines.append("")
            continue

        words = line.split() if " " in line else list(line)
        current = ""

        for word in words:
            test = word if current == "" else (f"{current} {word}" if " " in line else current + word)
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


def draw_number_badge(page, rect, number, severity):
    c = get_color(severity)
    size = 14
    cx = rect.x0
    cy = rect.y0
    page.draw_circle((cx, cy), size / 2, color=c["badge"], fill=c["badge"])
    page.insert_text(
        (cx - size / 2 + 2, cy - size / 2 + 2),
        str(number),
        fontsize=8,
        color=c["label_fg"],
    )


def annotate_pages(doc, items):
    for idx, item in enumerate(items, start=1):
        page_number = item.get("page_number", 1)
        bbox = item.get("bbox", [20, 20, 200, 60])
        page_width = item.get("page_width")
        page_height = item.get("page_height")
        severity = item.get("severity", "info")

        page_index = max(0, page_number - 1)
        if page_index >= len(doc):
            continue

        page = doc[page_index]
        rect = fitz.Rect(bbox[0], bbox[1], bbox[2], bbox[3])
        page_rect = page.rect

        if page_width and page_height:
            sx = page_rect.width / float(page_width)
            sy = page_rect.height / float(page_height)
            rect = fitz.Rect(rect.x0 * sx, rect.y0 * sy, rect.x1 * sx, rect.y1 * sy)
        else:
            if 0 <= rect.x0 <= 1.5 and 0 <= rect.y0 <= 1.5 and 0 <= rect.x1 <= 1.5 and 0 <= rect.y1 <= 1.5:
                rect = fitz.Rect(
                    rect.x0 * page_rect.width,
                    rect.y0 * page_rect.height,
                    rect.x1 * page_rect.width,
                    rect.y1 * page_rect.height,
                )

        c = get_color(severity)
        page.draw_rect(rect, color=c["border"], fill=c["highlight"], width=1.5)
        draw_number_badge(page, rect, idx, severity)


def render_review_page(items, page_width_pt, page_height_pt, font_path):
    scale = 2
    w = int(page_width_pt * scale)
    h = int(page_height_pt * scale)
    margin = int(30 * scale)

    image = Image.new("RGB", (w, h), (255, 255, 255))
    draw = ImageDraw.Draw(image)

    def load_font(size):
        if font_path.lower().endswith(".ttc"):
            return ImageFont.truetype(font_path, size, index=0)
        return ImageFont.truetype(font_path, size)

    title_font = load_font(int(16 * scale))
    body_font = load_font(int(9 * scale))
    small_font = load_font(int(8 * scale))

    y = margin

    draw.rectangle([0, 0, w, int(55 * scale)], fill=(38, 64, 115))
    draw.text((margin, int(17 * scale)), "レビュー指摘一覧", font=title_font, fill=(255, 255, 255))
    y = int(65 * scale)

    legend_labels = [("critical", "重大"), ("warning", "注意"), ("info", "情報")]
    lx = margin
    for sev, label in legend_labels:
        chip_color = get_color(sev)["pil_chip"]
        chip_bbox = draw.textbbox((0, 0), label, font=small_font)
        cw = chip_bbox[2] - chip_bbox[0] + int(10 * scale // 2)
        ch = chip_bbox[3] - chip_bbox[1] + int(6 * scale // 2)
        draw.rounded_rectangle([lx, y, lx + cw, y + ch], radius=int(3 * scale // 2), fill=chip_color)
        draw.text((lx + int(5 * scale // 2), y + int(3 * scale // 2)), label, font=small_font, fill=(255, 255, 255))
        lx += cw + int(8 * scale // 2)
    y += int(28 * scale)

    draw.line([(margin, y), (w - margin, y)], fill=(180, 180, 180), width=2)
    y += int(12 * scale)

    body_line_h = int(body_font.size * 1.35)
    small_line_h = int(small_font.size * 1.35)

    for idx, item in enumerate(items, start=1):
        if y > h - margin * 2:
            break

        severity = item.get("severity", "info")
        page_num = item.get("page_number", 1)
        message = item.get("message", "")
        suggestion = item.get("suggestion", "")
        c = get_color(severity)
        chip_color = c["pil_chip"]
        label = c["ja"] or severity

        badge_size = int(18 * scale)
        draw.ellipse([margin, y, margin + badge_size, y + badge_size], fill=chip_color)
        num_text = str(idx)
        nb = draw.textbbox((0, 0), num_text, font=small_font)
        draw.text(
            (margin + (badge_size - (nb[2] - nb[0])) // 2, y + (badge_size - (nb[3] - nb[1])) // 2),
            num_text,
            font=small_font,
            fill=(255, 255, 255),
        )

        chip_pad = int(5 * scale // 2)
        tag_x = margin + badge_size + int(8 * scale)
        cb = draw.textbbox((0, 0), label, font=small_font)
        cw = cb[2] - cb[0] + chip_pad * 2
        ch = cb[3] - cb[1] + chip_pad * 2
        draw.rounded_rectangle([tag_x, y, tag_x + cw, y + ch], radius=int(3 * scale // 2), fill=chip_color)
        draw.text((tag_x + chip_pad, y + chip_pad), label, font=small_font, fill=(255, 255, 255))

        page_text = f"  ページ {page_num}"
        draw.text((tag_x + cw + int(8 * scale // 2), y + chip_pad), page_text, font=small_font, fill=(120, 120, 120))
        y += max(badge_size, ch) + int(6 * scale)

        text_x = margin + int(10 * scale)
        for line in wrap_text(draw, message, body_font, w - text_x - margin):
            if y > h - margin:
                break
            draw.text((text_x, y), line, font=body_font, fill=(20, 20, 20))
            y += body_line_h

        if suggestion:
            for line in wrap_text(draw, "改善案: " + suggestion, small_font, w - text_x - margin - int(16 * scale)):
                if y > h - margin:
                    break
                draw.text((text_x + int(16 * scale), y), line, font=small_font, fill=(60, 100, 60))
                y += small_line_h

        y += int(10 * scale)
        draw.line([(margin, y), (w - margin, y)], fill=(220, 220, 220), width=1)
        y += int(10 * scale)

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

    if items:
        annotate_pages(doc, items)
        first_page = doc[0]
        page_width = first_page.rect.width
        page_height = first_page.rect.height
        new_page = doc.new_page(-1, width=page_width, height=page_height)
        review_image = render_review_page(items, page_width, page_height, font_path)
        new_page.insert_image(
            fitz.Rect(0, 0, page_width, page_height),
            stream=review_image,
            keep_proportion=False,
            overlay=True,
        )

    doc.save(args.output, garbage=4, deflate=True)
    doc.close()


if __name__ == "__main__":
    main()
