#!/usr/bin/env python3
"""Build a self-contained review page for the 2026-10-10 23:00 Hero audit."""

from __future__ import annotations

import argparse
import base64
import hashlib
import html
import json
from pathlib import Path


RECOGNITION_GROUPS = [
    {
        "title": "非有效手牌误判为 Hero",
        "severity": "critical",
        "indices": [1],
        "comparison_indices": [],
        "summary": (
            "画面没有 Hero 底牌，识别却给出 is_hero_turn=true；Hero cards 为空且 "
            "stack=0。ddbaby 请求校验已拒绝，客户端无结果。"
        ),
    },
    {
        "title": "空座误判 active：23:50 片段",
        "severity": "critical",
        "indices": list(range(17, 25)),
        "comparison_indices": [],
        "summary": (
            "右侧带“+”的空座被识别为 active/stack=0，造成活跃人数与位置整体偏移。"
            "8次响应均不能作为可信策略样本，4个 cycle 的首次展示使用了错误输入。"
        ),
    },
    {
        "title": "空座误判 active：00:04 片段",
        "severity": "critical",
        "indices": list(range(83, 89)),
        "comparison_indices": [],
        "summary": (
            "Hero 实际投入 BB 2，左下玩家有明确 STR 4；空座进入位置构造后，Hero 被映射为 "
            "straddle。前4帧返回面对下注的非法 check 并被服务端拦截，后2帧 fold 虽合法但输入位置不可信。"
        ),
    },
    {
        "title": "公共牌 7♥ 短暂识别为 9♥",
        "severity": "warning",
        "indices": [45],
        "comparison_indices": [46],
        "summary": (
            "23:54:40 原图牌面为 T♠ 4♠ 7♥，识别成 T♠ 4♠ 9♥；下一帧恢复。"
            "两帧均返回 check，但首个展示使用了错误牌面。"
        ),
    },
    {
        "title": "公共牌 7♣ 短暂识别为 9♣",
        "severity": "warning",
        "indices": [145],
        "comparison_indices": [146],
        "summary": (
            "00:13:48 原图牌面为 K♠ 4♠ 7♣，识别成 K♠ 4♠ 9♣；下一帧恢复。"
            "两帧均返回 check，但首个展示使用了错误牌面。"
        ),
    },
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--audit", required=True, type=Path)
    parser.add_argument("--images-dir", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    return parser.parse_args()


def load_json(path: Path) -> dict:
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


def escaped(value: object) -> str:
    return html.escape(str(value), quote=True)


def json_block(value: object) -> str:
    rendered = json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True)
    return html.escape(rendered)


def image_figure(images_dir: Path, item: dict, caption_prefix: str = "") -> str:
    image_name = item["image"]
    image_path = safe_image_path(images_dir, image_name)
    image_data = base64.b64encode(image_path.read_bytes()).decode("ascii")
    caption = f"{caption_prefix}{image_name}"
    return (
        f'<figure><img loading="lazy" src="data:image/jpeg;base64,{image_data}" '
        f'alt="{escaped(caption)}"><figcaption>{escaped(caption)}</figcaption></figure>'
    )


def compact_item(item: dict) -> dict:
    return {
        "index": item["index"],
        "image": item["image"],
        "trace_id": item["trace_id"],
        "cycle_id": item["cycle_id"],
        "recognition_assessment": item["recognition_assessment"],
        "recognition": item["recognition"],
        "decision_assessment": item["decision_assessment"],
        "response_accepted": item["response_accepted"],
        "decision": item["decision"],
        "first_display_ms": item["first_display_ms"],
    }


def recognition_cards(audit: dict, images_dir: Path) -> str:
    by_index = {item["index"]: item for item in audit["images"]}
    cards = []
    for group in RECOGNITION_GROUPS:
        issue_items = [by_index[index] for index in group["indices"]]
        comparison_items = [by_index[index] for index in group["comparison_indices"]]
        figures = [image_figure(images_dir, item) for item in issue_items]
        figures.extend(
            image_figure(images_dir, item, "对照下一帧 · ")
            for item in comparison_items
        )
        all_details = [compact_item(item) for item in issue_items + comparison_items]
        cards.append(
            f"""
            <article class="issue {escaped(group['severity'])}">
              <div class="issue-head"><span class="badge">确认的识别问题</span>
                <span>{len(issue_items)} 张问题图</span></div>
              <h3>{escaped(group['title'])}</h3>
              <p>{escaped(group['summary'])}</p>
              <div class="image-grid">{''.join(figures)}</div>
              <details><summary>逐图识别与决策索引</summary><pre>{json_block(all_details)}</pre></details>
            </article>
            """
        )
    return "".join(cards)


def solver_cards(audit: dict, images_dir: Path) -> str:
    groups: dict[str, list[dict]] = {}
    for item in audit["images"]:
        if item["decision_assessment"] != "solver_sensitive":
            continue
        state = item["decision"]["state_sha256"]
        groups.setdefault(state, []).append(item)

    review_notes = [
        "5♠7♦ 在 4♣ 2♥ A♠、pot=134 面对54选择 call；仅弱听牌，需完整范围、人数和 rake 复核。",
        "A♥9♥ 在 Q♣ 3♠ 6♥ 3♥、pot=41 面对14选择 raise 到42；半诈唬可成立，但频率与尺度依赖范围。",
        "T♦4♠ 在 7♦ 9♣ 4♣、pot=90 面对18选择 call；多人池底对继续需要独立EV校验。",
        "A♣Q♦ 在 K♠ 4♠ 7♣、pot=85 面对34选择 call；无同花听牌的弱继续需要独立EV校验。",
    ]
    ordered = sorted(groups.values(), key=lambda items: items[0]["index"])
    if len(ordered) != len(review_notes):
        raise ValueError(
            f"expected {len(review_notes)} solver-sensitive states, got {len(ordered)}"
        )
    cards = []
    for number, (items, note) in enumerate(zip(ordered, review_notes), start=1):
        first = items[0]
        decision = first["decision"]
        cards.append(
            f"""
            <article class="issue review">
              <div class="issue-head"><span class="badge">待 Solver 复核</span>
                <span>#{number} · {len(items)} 次响应</span></div>
              <h3>{escaped(' / '.join(first['recognition']['hero_cards']))} · {escaped(decision['action'])}</h3>
              <p>{escaped(note)}</p>
              <div class="image-grid single">{image_figure(images_dir, first)}</div>
              <details><summary>该状态全部逐图索引</summary><pre>{json_block([compact_item(item) for item in items])}</pre></details>
            </article>
            """
        )
    return "".join(cards)


def build_page(audit: dict, images_dir: Path) -> str:
    recognition = audit["recognition"]
    decision = audit["decision"]
    cycles = audit["cycles"]
    latency = audit["latency_ms"]["hero_turn_to_first_display"]
    grouped_problem_indices = {
        index for group in RECOGNITION_GROUPS for index in group["indices"]
    }
    actual_problem_indices = {
        item["index"]
        for item in audit["images"]
        if item["recognition_assessment"] != "no_obvious_core_field_error"
    }
    if grouped_problem_indices != actual_problem_indices:
        raise ValueError("recognition page groups do not match audit problem images")
    if len(actual_problem_indices) != recognition["confirmed_error_images"]:
        raise ValueError("recognition problem count does not match audit summary")
    return f"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>10月10日23点起 Hero 全链路审计</title>
  <style>
    :root {{ color-scheme:dark; font-family:Inter,system-ui,-apple-system,"PingFang SC",sans-serif; background:#07110e; color:#edf7f1; }}
    * {{ box-sizing:border-box; }} body {{ margin:0; background:radial-gradient(circle at top,#123328 0,#07110e 38%); }}
    main {{ width:min(1180px,94vw); margin:0 auto; padding:42px 0 90px; }}
    h1 {{ margin:0 0 10px; font-size:clamp(28px,5vw,48px); }} h2 {{ margin-top:44px; }} h3 {{ margin:14px 0 8px; }}
    p,li {{ line-height:1.75; }} .lead,.muted {{ color:#b7cabf; }}
    .summary {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(190px,1fr)); gap:12px; margin:28px 0; }}
    .metric,.issue,.panel {{ background:#10221b; border:1px solid #29483b; border-radius:16px; padding:20px; box-shadow:0 12px 35px #0004; }}
    .metric strong {{ display:block; color:#82e7af; font-size:31px; }} .metric span {{ color:#b7cabf; }}
    .alert {{ border-color:#7d3f36; background:#251713; }} .alert strong {{ color:#ff9d82; }}
    .issues {{ display:grid; gap:24px; }} .issue-head {{ display:flex; flex-wrap:wrap; gap:10px; align-items:center; color:#b7cabf; }}
    .badge {{ border-radius:999px; padding:5px 10px; font-weight:750; color:#07110e; background:#ff9d82; }}
    .warning .badge {{ background:#ffd271; }} .review .badge {{ background:#8cc8ff; }}
    .image-grid {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(235px,1fr)); gap:12px; margin:18px 0; }}
    .image-grid.single {{ grid-template-columns:minmax(240px,420px); }} figure {{ margin:0; }}
    img {{ display:block; width:100%; max-height:760px; object-fit:contain; border-radius:11px; background:#000; }}
    figcaption {{ margin-top:6px; color:#8eaa9b; font-size:12px; overflow-wrap:anywhere; }}
    table {{ width:100%; border-collapse:collapse; background:#0b1914; border-radius:12px; overflow:hidden; }}
    th,td {{ padding:11px 12px; border-bottom:1px solid #29483b; text-align:left; }} th {{ color:#8edfb1; }}
    details {{ margin-top:12px; border-top:1px solid #29483b; padding-top:11px; }} summary {{ cursor:pointer; color:#8edfb1; }}
    pre {{ white-space:pre-wrap; overflow-wrap:anywhere; background:#07110e; border-radius:9px; padding:14px; font-size:12px; max-height:540px; overflow:auto; }}
    nav {{ display:flex; flex-wrap:wrap; gap:10px; margin:20px 0 30px; }} nav a {{ color:#07110e; background:#8edfb1; border-radius:999px; padding:8px 13px; text-decoration:none; font-weight:700; }}
    footer {{ margin-top:42px; color:#8eaa9b; }}
  </style>
</head>
<body><main>
  <h1>Hero 识别、决策与耗时审计</h1>
  <p class="lead">窗口：2026-10-10 23:00 至 2026-10-11 14:10。实际150张 Hero 图片集中在23:46:56至00:14:02。本页展示确认的问题原图、决策分母和首次结果耗时；没有把动作合法等同于GTO最优。</p>
  <nav><a href="#recognition">识别问题</a><a href="#decision">决策评估</a><a href="#latency">耗时</a><a href="#boundary">口径边界</a></nav>
  <section class="summary">
    <div class="metric"><strong>{recognition['images']}</strong><span>Hero 图片</span></div>
    <div class="metric alert"><strong>{recognition['confirmed_error_images']}</strong><span>确认识别问题图片</span></div>
    <div class="metric"><strong>{decision['correct_input_states']}</strong><span>正确输入独立决策状态</span></div>
    <div class="metric"><strong>{decision['solver_sensitive_states']}</strong><span>待 Solver 复核状态</span></div>
    <div class="metric"><strong>{cycles['first_display_complete']}/{cycles['total']}</strong><span>周期有首次展示</span></div>
    <div class="metric"><strong>{latency['mean'] / 1000:.3f}s</strong><span>首次展示平均耗时</span></div>
  </section>

  <section class="panel">
    <h2>审计结论</h2>
    <ul>
      <li>133/150张未发现核心可见字段明显不一致；17张确认问题：14张空座误判active、2张公共牌7→9、1张false Hero。</li>
      <li>149个原始响应中145个通过服务端校验；排除12个错误输入后，决策审计分母为133次响应、59个独立状态。</li>
      <li>正确输入的133次响应均满足动作合法性，未确认新的确定性决策错误；4状态/11次响应单列待solver复核。</li>
      <li>64个首次展示中有7个使用确认错误的输入；“有结果”和“输入可信”必须分开统计。</li>
    </ul>
  </section>

  <h2 id="recognition">确认的识别问题</h2>
  <div class="issues">{recognition_cards(audit, images_dir)}</div>

  <h2 id="decision">决策评估</h2>
  <section class="panel">
    <table><thead><tr><th>阶段</th><th>响应次数</th><th>独立状态</th></tr></thead><tbody>
      <tr><td>GTO原始响应</td><td>{decision['raw_responses']}</td><td>{decision['raw_states']}</td></tr>
      <tr><td>响应校验通过</td><td>{decision['accepted_responses']}</td><td>{decision['accepted_states']}</td></tr>
      <tr><td>排除确认错误输入</td><td>{decision['correct_input_responses']}</td><td>{decision['correct_input_states']}</td></tr>
      <tr><td>未发现明显规则/策略问题</td><td>{decision['no_obvious_issue_responses']}</td><td>{decision['no_obvious_issue_states']}</td></tr>
      <tr><td>待 Solver 复核</td><td>{decision['solver_sensitive_responses']}</td><td>{decision['solver_sensitive_states']}</td></tr>
      <tr><td>正确输入下确认错误</td><td>{decision['confirmed_error_correct_input_responses']}</td><td>{decision['confirmed_error_correct_input_states']}</td></tr>
    </tbody></table>
    <p class="muted">00:04:41片段的4次非法check来自错误座位输入，已被ddbaby响应校验器拦截，未作为客户端有效决策，也未放入正确输入分母。</p>
  </section>
  <div class="issues">{solver_cards(audit, images_dir)}</div>

  <h2 id="latency">全链路耗时</h2>
  <section class="panel">
    <p>67个客户端周期中64个有首次展示、3个无结果。首屏平均 <strong>{latency['mean']:.1f}ms</strong>，P50 {latency['p50']:.0f}ms，P95 {latency['p95']:.1f}ms，最大 {latency['max']}ms；2/64超过3秒，0个超过4秒。</p>
    <table><thead><tr><th>分段</th><th>平均</th><th>P50</th><th>P95</th><th>最大</th></tr></thead><tbody>
      <tr><td>Hero检测到采集</td><td>92ms</td><td>0ms</td><td>508ms</td><td>1,580ms</td></tr>
      <tr><td>采集到HTTP开始</td><td>123ms</td><td>118ms</td><td>172ms</td><td>200ms</td></tr>
      <tr><td>服务端识别</td><td>1,607ms</td><td>1,553ms</td><td>2,243ms</td><td>2,445ms</td></tr>
      <tr><td>服务端决策</td><td>82ms</td><td>60ms</td><td>212ms</td><td>791ms</td></tr>
      <tr><td>服务端外HTTP/网络</td><td>155ms</td><td>148ms</td><td>237ms</td><td>285ms</td></tr>
      <tr><td>响应到展示</td><td>15ms</td><td>16ms</td><td>23ms</td><td>35ms</td></tr>
    </tbody></table>
    <p class="muted">平均分段为 92 + 123 + 1,690 + 155 + 15 = 2,075ms。识别平均1.607秒，占首屏77.4%，仍是主要耗时。queue backlog属于同一客户端前置时间轴内部指标，没有重复叠加。</p>
  </section>

  <h2 id="boundary">口径边界</h2>
  <section class="panel">
    <ul>
      <li>明显识别问题统计来自逐图核对核心可见字段，不是所有对手字段的逐像素真值标注。</li>
      <li>“未发现明显问题”不代表GTO最优；4个高风险状态保留为solver-sensitive。</li>
      <li>耗时主指标每个decision_cycle_id只取第一条client_complete，重复帧和重复展示不重复计数。</li>
      <li>本页是冻结窗口的审计证据；当前没有包含任何生产修复或部署结论。</li>
    </ul>
  </section>
  <footer>生成来源：脱敏 audit.json 与冻结原图。页面自包含，不依赖外部图片地址。</footer>
</main></body></html>"""


def main() -> None:
    args = parse_args()
    audit = load_json(args.audit)
    page = build_page(audit, args.images_dir)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(page, encoding="utf-8")
    payload = page.encode("utf-8")
    print(json.dumps({
        "output": str(args.output),
        "bytes": len(payload),
        "sha256": hashlib.sha256(payload).hexdigest(),
        "embedded_problem_images": 17,
        "embedded_comparison_images": 2,
        "embedded_solver_representatives": 4,
    }, ensure_ascii=False, sort_keys=True))


if __name__ == "__main__":
    main()
