#!/usr/bin/env python3
"""Build a self-contained review page for flagged green-table decision states."""

from __future__ import annotations

import argparse
import base64
import hashlib
import html
import json
from pathlib import Path


INCLUDED_CLASSIFICATIONS = {
    "excluded_recognition_mismatch",
    "strict_free_fold_error",
    "strategy_bad_case",
    "solver_sensitive_review",
}
CLASSIFICATION_LABELS = {
    "excluded_recognition_mismatch": "识别错误",
    "strict_free_fold_error": "确定性动作错误",
    "strategy_bad_case": "策略 Bad Case",
    "solver_sensitive_review": "待 Solver 复核",
}
CLASSIFICATION_NOTES = {
    "excluded_recognition_mismatch": "原图可见下注与识别请求不一致；该状态不进入策略正确性分母。",
    "strict_free_fold_error": "Hero 无需补注即可 check，fold 被 check 严格支配。",
    "strategy_bad_case": "识别请求与原图一致，但返回动作存在明显策略问题。",
    "solver_sensitive_review": "动作是否合法不能证明其 GTO 最优，需要独立范围、rake、EV 和策略树复核。",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--audit", required=True, type=Path)
    parser.add_argument("--decisions", required=True, type=Path)
    parser.add_argument("--images-dir", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    return parser.parse_args()


def load_json(path: Path) -> object:
    with path.open("r", encoding="utf-8") as source:
        return json.load(source)


def safe_image_path(images_dir: Path, image_name: str) -> Path:
    if Path(image_name).name != image_name or not image_name.lower().endswith(".jpg"):
        raise ValueError(f"unsafe image name: {image_name!r}")
    root = images_dir.resolve(strict=True)
    candidate = (root / image_name).resolve(strict=True)
    if candidate.parent != root:
        raise ValueError(f"image escapes source directory: {image_name!r}")
    return candidate


def json_block(value: object) -> str:
    rendered = json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True)
    return html.escape(rendered)


def build_page(audit: dict, decisions: dict, images_dir: Path) -> tuple[str, dict[str, int]]:
    items_by_trace = {item["trace_id"]: item for item in audit["items"]}
    records_by_trace = {record["trace_id"]: record for record in decisions["records"]}
    states = [
        state
        for state in audit["decision"]["states"]
        if state["classification"] in INCLUDED_CLASSIFICATIONS
    ]
    states.sort(key=lambda state: (state["classification"], state["times"][0]))
    if not states:
        raise ValueError("no flagged decision states")

    cards: list[str] = []
    recognition_count = 0
    strict_count = 0
    strict_occurrences = 0
    strategy_count = 0
    strategy_occurrences = 0
    solver_count = 0
    for index, state in enumerate(states, start=1):
        classification = state["classification"]
        recognition_count += classification == "excluded_recognition_mismatch"
        strict_count += classification == "strict_free_fold_error"
        if classification == "strict_free_fold_error":
            strict_occurrences += state["occurrences"]
        strategy_count += classification == "strategy_bad_case"
        if classification == "strategy_bad_case":
            strategy_occurrences += state["occurrences"]
        solver_count += classification == "solver_sensitive_review"
        trace_id = state["trace_ids"][0]
        item = items_by_trace[trace_id]
        record = records_by_trace[trace_id]
        image_trace_ids = (state["trace_ids"]
                           if classification == "excluded_recognition_mismatch"
                           else [trace_id])
        image_tags = []
        for image_trace_id in image_trace_ids:
            image_item = items_by_trace[image_trace_id]
            image_path = safe_image_path(images_dir, image_item["image"])
            image_data = base64.b64encode(image_path.read_bytes()).decode("ascii")
            image_tags.append(
                f'<figure><img src="data:image/jpeg;base64,{image_data}" '
                f'alt="{html.escape(image_item["image"])}">'
                f'<figcaption>{html.escape(image_item["image"])}</figcaption></figure>')
        cards.append(
            f"""
            <article class="issue {html.escape(classification)}">
              <div class="issue-head">
                <span class="badge">{html.escape(CLASSIFICATION_LABELS[classification])}</span>
                <span>#{index} · {html.escape(state['stage'])} · {html.escape(state['position'])}</span>
              </div>
              <h2>{html.escape(' '.join(state['hero_cards']))} · {html.escape(state['action'])}</h2>
              <p>{html.escape(state.get('recognition_mismatch') or state.get('review_note') or CLASSIFICATION_NOTES[classification])}</p>
              <dl>
                <div><dt>出现时间</dt><dd>{html.escape('、'.join(state['times']))}</dd></div>
                <div><dt>出现次数</dt><dd>{state['occurrences']}</dd></div>
                <div><dt>Pot / Hero bet / To call</dt><dd>{state['pot']} / {state['hero_bet']} / {state['amount_to_call']}</dd></div>
                <div><dt>Trace</dt><dd>{html.escape('、'.join(state['trace_ids']))}</dd></div>
              </dl>
              <div class="image-grid">{''.join(image_tags)}</div>
              <details><summary>完整决策请求</summary><pre>{json_block(record['request'])}</pre></details>
              <details><summary>原始决策响应</summary><pre>{json_block(record['response'])}</pre></details>
              <details><summary>识别结果</summary><pre>{json_block(record['recognition'])}</pre></details>
            </article>
            """
        )

    page = f"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>2026-10-10 绿桌 Hero 决策问题复核</title>
  <style>
    :root {{ color-scheme: dark; font-family: system-ui, sans-serif; background:#07110e; color:#ecf5ef; }}
    body {{ margin:0; }} main {{ width:min(1120px,92vw); margin:0 auto; padding:40px 0 80px; }}
    h1 {{ margin-bottom:8px; }} .lead {{ color:#b7c9bf; line-height:1.7; }}
    .summary {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(180px,1fr)); gap:12px; margin:28px 0; }}
    .metric,.issue {{ background:#10221b; border:1px solid #29483b; border-radius:14px; padding:20px; }}
    .metric strong {{ display:block; font-size:30px; color:#7fe0aa; }} .metric span {{ color:#b7c9bf; }}
    .issues {{ display:grid; gap:24px; }} .issue-head {{ display:flex; gap:10px; align-items:center; color:#b7c9bf; }}
    .badge {{ border-radius:999px; padding:5px 10px; font-weight:700; color:#07110e; background:#ffcb6b; }}
    .excluded_recognition_mismatch .badge {{ background:#ff9f68; }}
    .strict_free_fold_error .badge,.strategy_bad_case .badge {{ background:#ff7777; }} h2 {{ margin:14px 0 6px; }}
    dl {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(230px,1fr)); gap:10px; }}
    dl div {{ background:#0a1813; border-radius:9px; padding:10px; }} dt {{ color:#8eaa9b; font-size:13px; }} dd {{ margin:5px 0 0; overflow-wrap:anywhere; }}
    .image-grid {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(260px,1fr)); gap:12px; margin:16px 0; }}
    figure {{ margin:0; }} img {{ display:block; width:100%; border-radius:10px; background:#000; }}
    figcaption {{ margin-top:6px; color:#8eaa9b; font-size:12px; overflow-wrap:anywhere; }}
    details {{ margin-top:10px; border-top:1px solid #29483b; padding-top:10px; }} summary {{ cursor:pointer; color:#8edfb1; }}
    pre {{ white-space:pre-wrap; overflow-wrap:anywhere; background:#07110e; padding:14px; border-radius:8px; font-size:12px; }}
    footer {{ margin-top:30px; color:#8eaa9b; font-size:13px; line-height:1.6; }}
  </style>
</head>
<body><main>
  <h1>绿桌 Hero 决策问题复核</h1>
  <p class="lead">范围：2026-10-10 的 138 张绿桌 Hero 样本。页面收录关键识别错误、已确认决策 bad case 和待 Solver 复核状态；图片、请求与响应均来自同一 trace。</p>
  <section class="summary">
    <div class="metric"><strong>{recognition_count}</strong><span>关键识别错误状态</span></div>
    <div class="metric"><strong>{strict_count}</strong><span>确定性 free-fold 错误状态</span></div>
    <div class="metric"><strong>{strategy_count}</strong><span>明显策略 bad case 状态</span></div>
    <div class="metric"><strong>{strict_occurrences + strategy_occurrences}</strong><span>已确认错误响应次数</span></div>
    <div class="metric"><strong>{solver_count}</strong><span>待 Solver 复核状态</span></div>
    <div class="metric"><strong>138</strong><span>绿桌专项样本</span></div>
  </section>
  <section class="issues">{''.join(cards)}</section>
  <footer>判定边界：识别错误不进入策略分母；free-fold 是确定性规则错误；9♠9♣ 是识别正确后的明显策略 bad case；solver-sensitive 状态没有被判定为错误或正确。兜底由 ddbaby 在校验合法响应后执行，客户端不参与动作改写。</footer>
</main></body></html>"""
    return page, {
        "recognition_states": recognition_count,
        "strict_states": strict_count,
        "strategy_bad_case_states": strategy_count,
        "confirmed_error_occurrences": strict_occurrences + strategy_occurrences,
        "solver_sensitive_states": solver_count,
    }


def main() -> None:
    args = parse_args()
    audit = load_json(args.audit)
    decisions = load_json(args.decisions)
    page, counts = build_page(audit, decisions, args.images_dir)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(page, encoding="utf-8")
    digest = hashlib.sha256(page.encode("utf-8")).hexdigest()
    print(json.dumps({
        "output": str(args.output),
        "sha256": digest,
        "bytes": len(page.encode("utf-8")),
        **counts,
    }, ensure_ascii=False, sort_keys=True))


if __name__ == "__main__":
    main()
