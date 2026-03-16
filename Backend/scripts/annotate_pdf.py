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


SEVERITY_CONFIG = {
    "critical": ("重大", (211, 47, 47)),
    "warning": ("注意", (237, 108, 2)),
    "info": ("情報", (2, 136, 209)),
}


def render_review_page(items, page_width_pt, page_height_pt, font_path):
    scale = 2
    w = int(page_width_pt * scale)
    h = int(page_height_pt * scale)
    margin = int(30 * scale)

    image = Image.new("RGB", (w, h), (255, 255, 255))
    draw = ImageDraw.Draw(image)

    title_font_size = int(16 * scale)
    body_font_size = int(9 * scale)
    small_font_size = int(8 * scale)

    def load_font(size):
        if font_path.lower().endswith(".ttc"):
            return ImageFont.truetype(font_path, size, index=0)
        return ImageFont.truetype(font_path, size)

    title_font = load_font(title_font_size)
    body_font = load_font(body_font_size)
    small_font = load_font(small_font_size)

    y = margin

    # Title
    draw.text((margin, y), "指摘事項", font=title_font, fill=(20, 20, 20))
    y += int(title_font_size * 1.6)
    draw.line([(margin, y), (w - margin, y)], fill=(180, 180, 180), width=2 * scale // 2)
    y += int(12 * scale // 2)

    for item in items:
        if y > h - margin * 2:
            break

        severity = item.get("severity", "info")
        label, chip_color = SEVERITY_CONFIG.get(severity, ("情報", (2, 136, 209)))
        page_num = item.get("page_number", 1)
        message = item.get("message", "")
        suggestion = item.get("suggestion", "")

        # Severity chip
        chip_pad_x = int(8 * scale // 2)
        chip_pad_y = int(4 * scale // 2)
        chip_text_bbox = draw.textbbox((0, 0), label, font=small_font)
        chip_w = chip_text_bbox[2] - chip_text_bbox[0] + chip_pad_x * 2
        chip_h = chip_text_bbox[3] - chip_text_bbox[1] + chip_pad_y * 2
        draw.rounded_rectangle(
            [margin, y, margin + chip_w, y + chip_h],
            radius=int(3 * scale // 2),
            fill=chip_color,
        )
        draw.text((margin + chip_pad_x, y + chip_pad_y), label, font=small_font, fill=(255, 255, 255))

        # Page number
        page_text = f"ページ {page_num}"
        draw.text((margin + chip_w + int(8 * scale // 2), y + chip_pad_y), page_text, font=small_font, fill=(120, 120, 120))
        y += chip_h + int(6 * scale // 2)

        # Message
        msg_lines = wrap_text(draw, message, body_font, w - margin * 2)
        for line in msg_lines:
            if y > h - margin:
                break
            draw.text((margin, y), line, font=body_font, fill=(20, 20, 20))
            y += int(body_font_size * 1.35)

        # Suggestion
        if suggestion:
            sug_prefix = "改善案: "
            sug_lines = wrap_text(draw, sug_prefix + suggestion, small_font, w - margin * 2 - int(16 * scale // 2))
            for i, line in enumerate(sug_lines):
                if y > h - margin:
                    break
                draw.text((margin + int(16 * scale // 2), y), line, font=small_font, fill=(80, 80, 80))
                y += int(small_font_size * 1.35)

        y += int(10 * scale // 2)
        draw.line([(margin, y), (w - margin, y)], fill=(220, 220, 220), width=1)
        y += int(10 * scale // 2)

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

    # Highlight referenced text blocks on each page (no callout)
    for item in items:
        page_number = item.get("page_number", 1)
        bbox = item.get("bbox", [20, 20, 200, 60])
        page_width = item.get("page_width")
        page_height = item.get("page_height")

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

        severity = item.get("severity", "info")
        _, chip_color = SEVERITY_CONFIG.get(severity, ("情報", (2, 136, 209)))
        border_color = tuple(c / 255.0 for c in chip_color)
        fill_color = tuple(c / 255.0 * 0.15 + 0.85 for c in chip_color)

        page.draw_rect(rect, color=border_color, fill=fill_color, width=1.5)

    # Add review summary page at the end
    if items:
        # Use same width as first page, A4 height
        first_page = doc[0]
        page_w = first_page.rect.width
        page_h = first_page.rect.height
        new_page = doc.new_page(-1, width=page_w, height=page_h)
        review_image = render_review_page(items, page_w, page_h, font_path)
        new_page.insert_image(
            fitz.Rect(0, 0, page_w, page_h),
            stream=review_image,
            keep_proportion=False,
            overlay=True,
        )

    doc.save(args.output, garbage=4, deflate=True)
    doc.close()


if __name__ == "__main__":
    main()
