#!/usr/bin/env python3
import argparse
import contextlib
import io
import json
import os

import fitz  # PyMuPDF
from paddleocr import PaddleOCR


def ocr_page(ocr, pix):
    img_bytes = pix.tobytes("png")
    with contextlib.redirect_stdout(io.StringIO()):
        result = ocr.ocr(img_bytes, cls=True)
    blocks = []
    if not result or not result[0]:
        return blocks
    for idx, line in enumerate(result[0]):
        bbox, (text, _score) = line
        x_coords = [p[0] for p in bbox]
        y_coords = [p[1] for p in bbox]
        blocks.append({
            "block_index": idx,
            "text": text,
            "bbox": [min(x_coords), min(y_coords), max(x_coords), max(y_coords)],
        })
    return blocks


def process_pdf(path, ocr):
    doc = fitz.open(path)
    pages = []
    for page_index in range(len(doc)):
        page = doc[page_index]
        pix = page.get_pixmap(dpi=150)
        blocks = ocr_page(ocr, pix)
        pages.append({
            "page_number": page_index + 1,
            "width": pix.width,
            "height": pix.height,
            "blocks": blocks,
        })
    return pages


def process_image(path, ocr):
    pix = fitz.Pixmap(path)
    blocks = ocr_page(ocr, pix)
    return [{
        "page_number": 1,
        "width": pix.width,
        "height": pix.height,
        "blocks": blocks,
    }]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=False)
    args = parser.parse_args()

    if not os.path.exists(args.input):
        raise SystemExit(f"input file not found: {args.input}")

    with contextlib.redirect_stdout(io.StringIO()):
        ocr = PaddleOCR(use_angle_cls=True, lang="japan", show_log=False)

    ext = os.path.splitext(args.input)[1].lower()
    if ext in [".pdf"]:
        pages = process_pdf(args.input, ocr)
    else:
        pages = process_image(args.input, ocr)

    payload = {"pages": pages}
    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            json.dump(payload, f, ensure_ascii=False)
    else:
        print(json.dumps(payload, ensure_ascii=False))


if __name__ == "__main__":
    main()
