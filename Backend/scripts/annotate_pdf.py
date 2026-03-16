#!/usr/bin/env python3
import argparse
import json
import os

import fitz  # PyMuPDF

SEVERITY_COLORS = {
    "critical": {
        "highlight": (1.0, 0.82, 0.82),
        "border":    (0.78, 0.15, 0.15),
        "label_bg":  (0.78, 0.15, 0.15),
        "label_fg":  (1.0, 1.0, 1.0),
        "badge":     (0.85, 0.15, 0.15),
        "ja":        "重大",
    },
    "warning": {
        "highlight": (1.0, 0.95, 0.75),
        "border":    (0.75, 0.50, 0.05),
        "label_bg":  (0.85, 0.60, 0.05),
        "label_fg":  (1.0, 1.0, 1.0),
        "badge":     (0.85, 0.55, 0.05),
        "ja":        "注意",
    },
    "info": {
        "highlight": (0.82, 0.92, 1.0),
        "border":    (0.18, 0.48, 0.78),
        "label_bg":  (0.18, 0.48, 0.78),
        "label_fg":  (1.0, 1.0, 1.0),
        "badge":     (0.18, 0.48, 0.78),
        "ja":        "情報",
    },
}
DEFAULT_COLOR = {
    "highlight": (0.90, 0.90, 0.90),
    "border":    (0.40, 0.40, 0.40),
    "label_bg":  (0.40, 0.40, 0.40),
    "label_fg":  (1.0, 1.0, 1.0),
    "badge":     (0.40, 0.40, 0.40),
    "ja":        "",
}


def color(c):
    """SEVERITY_COLORS エントリまたは DEFAULT_COLOR を返す。"""
    return SEVERITY_COLORS.get(c, DEFAULT_COLOR)


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
def draw_number_badge(page, rect, number, severity):
    """指摘箇所に番号バッジを描画する。"""
    c = color(severity)
    size = 14
    cx = rect.x0
    cy = rect.y0
    badge_rect = fitz.Rect(cx - size / 2, cy - size / 2, cx + size / 2, cy + size / 2)
    page.draw_circle((cx, cy), size / 2, color=c["badge"], fill=c["badge"])
    page.insert_text(
        (badge_rect.x0 + 2, badge_rect.y0 + 2),
        str(number),
        fontsize=8,
        color=c["label_fg"],
    )


def annotate_pages(doc, items):
    """各ページの指摘箇所をハイライト＋番号バッジで示す。"""
    for idx, item in enumerate(items, start=1):
        page_number = item.get("page_number", 1)
        bbox = item.get("bbox", [20, 20, 200, 60])
        page_width = item.get("page_width")
        page_height = item.get("page_height")
        severity = item.get("severity", "")

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
        c = color(severity)
        page.draw_rect(rect, color=c["border"], fill=c["highlight"], width=1.5)
        draw_number_badge(page, rect, idx, severity)


def add_summary_page(doc, items, font_path):
    """指摘内容の一覧ページを末尾に追加する。"""
    PAGE_W, PAGE_H = 595, 842  # A4
    MARGIN = 50
    LINE_H = 16
    ITEM_GAP = 10

    page = doc.new_page(width=PAGE_W, height=PAGE_H)

    # ---- ヘッダー ----
    page.draw_rect(
        fitz.Rect(0, 0, PAGE_W, 60),
        color=(0.15, 0.25, 0.45),
        fill=(0.15, 0.25, 0.45),
        width=0,
    )
    page.insert_text(
        (MARGIN, 38),
        "レビュー指摘一覧",
        fontsize=20,
        color=(1, 1, 1),
        fontname="helv",
    )

    # ---- 凡例 ----
    legend_y = 75
    page.insert_text((MARGIN, legend_y), "重要度:", fontsize=9, color=(0.3, 0.3, 0.3))
    lx = MARGIN + 48
    for sev, label in [("critical", "重大"), ("warning", "注意"), ("info", "情報")]:
        c = color(sev)
        page.draw_rect(
            fitz.Rect(lx, legend_y - 9, lx + 36, legend_y + 2),
            color=c["badge"], fill=c["badge"], width=0,
        )
        page.insert_text((lx + 4, legend_y), label, fontsize=8, color=(1, 1, 1))
        lx += 46

    # ---- 罫線 ----
    y = 95
    page.draw_line((MARGIN, y), (PAGE_W - MARGIN, y), color=(0.7, 0.7, 0.7), width=0.5)
    y += 12

    for idx, item in enumerate(items, start=1):
        severity = item.get("severity", "")
        page_num = item.get("page_number", "-")
        message = item.get("message", "")
        suggestion = item.get("suggestion", "")
        c = color(severity)
        sev_ja = c["ja"] or severity

        # ページ送り
        if y > PAGE_H - MARGIN - 60:
            page = doc.new_page(width=PAGE_W, height=PAGE_H)
            page.draw_rect(
                fitz.Rect(0, 0, PAGE_W, 60),
                color=(0.15, 0.25, 0.45),
                fill=(0.15, 0.25, 0.45),
                width=0,
            )
            page.insert_text((MARGIN, 38), "レビュー指摘一覧（続き）", fontsize=20, color=(1, 1, 1))
            y = 75

        # ---- 番号バッジ + 重要度タグ + ページ番号 ----
        badge_r = fitz.Rect(MARGIN, y, MARGIN + 18, y + 14)
        page.draw_rect(badge_r, color=c["badge"], fill=c["badge"], width=0)
        page.insert_text((MARGIN + 3, y + 11), str(idx), fontsize=8, color=(1, 1, 1))

        tag_x = MARGIN + 24
        tag_r = fitz.Rect(tag_x, y, tag_x + 30, y + 14)
        page.draw_rect(tag_r, color=c["badge"], fill=c["badge"], width=0)
        page.insert_text((tag_x + 3, y + 11), sev_ja, fontsize=8, color=(1, 1, 1))

        page.insert_text(
            (tag_x + 36, y + 11),
            f"P{page_num}",
            fontsize=9,
            color=(0.4, 0.4, 0.4),
        )
        y += LINE_H + 2

        # ---- 指摘テキスト ----
        text_x = MARGIN + 10
        max_w = PAGE_W - MARGIN - text_x

        msg_rect = fitz.Rect(text_x, y, text_x + max_w, y + LINE_H * 4)
        page.insert_textbox(
            msg_rect,
            f"指摘: {message}",
            fontsize=10,
            color=(0.1, 0.1, 0.1),
            fontname="helv",
        )
        # テキストの実際の高さを推定（1行16pt、最大4行）
        approx_lines = max(1, len(message) // 38 + 1)
        y += min(approx_lines, 4) * LINE_H + 2

        if suggestion:
            sug_rect = fitz.Rect(text_x, y, text_x + max_w, y + LINE_H * 4)
            page.insert_textbox(
                sug_rect,
                f"改善案: {suggestion}",
                fontsize=10,
                color=(0.2, 0.4, 0.2),
                fontname="helv",
            )
            approx_lines_sug = max(1, len(suggestion) // 38 + 1)
            y += min(approx_lines_sug, 4) * LINE_H + 2

        # ---- 区切り線 ----
        y += ITEM_GAP
        page.draw_line(
            (MARGIN, y), (PAGE_W - MARGIN, y),
            color=(0.85, 0.85, 0.85), width=0.5,
        )
        y += 8


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
        add_summary_page(doc, items, font_path)

    doc.save(args.output, garbage=4, deflate=True)
    doc.close()


if __name__ == "__main__":
    main()
