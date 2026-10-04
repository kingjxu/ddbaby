#!/usr/bin/env python3
"""Build a self-contained HTML evidence page for flagged strategy states."""

import argparse
import base64
import html
import json
from pathlib import Path


CLASS_META = {
    "strict_free_fold": ("明确错误", "strict"),
    "high_risk_preflop": ("高风险 · Preflop", "risk"),
    "high_risk_multiway_postflop": ("高风险 · 多人底池", "risk"),
    "solver_sensitive_river_bluff": ("需 Solver 复核", "solver"),
}

CASE_NOTES = {
    "2026-10-03 18:26:35-hero.jpg": {
        "title": "SB ATo 低价跟注却 100% fold",
        "reason": "Hero 只需补 3，当前 pot=19，即时底池赔率约 13.64%；均匀随机三手牌 Monte Carlo 摊分胜率约 33.95%。结果很可疑，但仍需真实范围求解。",
        "next": "用相同 ante/straddle/rake 和四人范围树复核 fold、call、raise 的 EV。",
    },
    "2026-10-03 18:31:54-hero.jpg": {
        "title": "UTG ATh 面对 raise + call 仍 100% call",
        "reason": "Hero 已 limp 4，BB 加注到 70、straddle 跟到 70；还需补 66，pot=167，且 CO/BTN 尚未完成行动。对收紧范围存在明显被支配和再加注风险。",
        "next": "按 BB 加注范围、straddle 跟注范围及身后行动者重建 preflop 多人决策树。",
    },
    "2026-10-03 18:37:31-hero.jpg": {
        "title": "KQo 河牌错失听牌后 0.68 pot all-in bluff",
        "reason": "Hero 在 flop raise 后 turn check，河牌错失 K♥ 高同花后门/听牌，下注 298 into 440。线路具备极化诈唬逻辑，不能仅凭绝对牌力判错。",
        "next": "用双方到河范围核对 K♥、Q♠ 的阻断价值、诈唬组合数和下注尺度 EV。",
    },
    "2026-10-03 18:46:11-hero.jpg": {
        "title": "T3o 已跟齐 straddle 却 100% fold",
        "reason": "Hero bet=4、全桌 max bet=4，不需再投入筹码即可 check。fold 立即放弃底池，check 保留获胜可能，因此 fold 被严格支配。",
        "next": "决策算法应保证 facing amount=0 时 fold 不在可选动作集合中。",
    },
    "2026-10-03 18:47:39-hero.jpg": {
        "title": "K6o 河牌 1.30 pot bluff",
        "reason": "Hero flop 以 6 持开放式顺子听牌继续，turn/river 错失后下注 87 into 67；6♥ 具有部分阻断逻辑，但超池下注频率需范围求解。",
        "next": "核对 6♥ 对对手跟注/弃牌范围的阻断效果及 1.30 pot 极化尺度的组合配比。",
    },
    "2026-10-03 18:48:26-hero.jpg": {
        "title": "BTN 93o 面对两个 limper 仍 85% raise",
        "reason": "Hero 持 9♣3♦，UTG+1 与 MP 已 limp 4，身后盲注和 straddle 仍在局；原策略 85% raise 到 15。对五手均匀随机牌的摊分胜率约 10.20%。",
        "next": "重点核对多人隔离范围、弃牌率假设和 93o 是否被错误纳入高频 bluff bucket。",
    },
    "2026-10-03 18:57:00-hero.jpg": {
        "title": "八人 flop 弱底对 75.7% call",
        "reason": "A♠ J♠ 6♥，Hero 9♦6♠，只有弱底对和低黑桃后门；面对 40 into 120，后面仍有 CO、BTN，前位过牌者也可能 check-raise。",
        "next": "用八人继续范围评估 25% 即时赔率、反向隐含赔率、身后 overcall/check-raise 风险。",
    },
    "2026-10-03 18:59:17-hero.jpg": {
        "title": "76o 已跟齐 straddle 却 100% fold",
        "reason": "Hero bet=4、全桌 max bet=4，可以免费 check 看 flop。与牌力、范围、rake 无关，fold 被 check 严格支配。",
        "next": "决策算法应保证 facing amount=0 时 fold 不在可选动作集合中。",
    },
}

SUITS = {"S": "♠", "H": "♥", "D": "♦", "C": "♣"}


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--audit", required=True)
    parser.add_argument("--requests", required=True)
    parser.add_argument("--replay", required=True)
    parser.add_argument("--images-dir", required=True)
    parser.add_argument("--output", required=True)
    return parser.parse_args()


def load_json(path):
    with open(path, "r", encoding="utf-8") as source:
        return json.load(source)


def load_json_lines(path):
    with open(path, "r", encoding="utf-8") as source:
        return [json.loads(line) for line in source if line.strip()]


def card_label(card):
    parts = card.split("-")
    if len(parts) != 2:
        return card
    return parts[0] + SUITS.get(parts[1], parts[1])


def safe_image_path(images_dir, filename):
    if Path(filename).name != filename or not filename.lower().endswith(".jpg"):
        raise ValueError("unsafe image filename: %r" % filename)
    root = images_dir.resolve()
    image = (root / filename).resolve()
    if image.parent != root:
        raise ValueError("image escapes images directory: %r" % filename)
    if not image.is_file():
        raise FileNotFoundError(str(image))
    return image


def esc(value):
    return html.escape(str(value), quote=True)


def pretty_json(value):
    return esc(json.dumps(value, ensure_ascii=False, indent=2))


def render_history(history):
    if not history:
        return '<p class="empty">无已记录 action_history</p>'
    rows = []
    for item in history:
        rows.append(
            "<tr><td>{}</td><td>{}</td><td>{}</td><td>{}</td></tr>".format(
                esc(item.get("stage", "")),
                esc(item.get("position", "")),
                esc(item.get("action", "")),
                esc(item.get("amount", 0)),
            )
        )
    return (
        '<table><thead><tr><th>街道</th><th>位置</th><th>动作</th><th>目标额</th></tr></thead>'
        "<tbody>{}</tbody></table>".format("".join(rows))
    )


def render_case(index, state, request, response, images_dir):
    filename = state["files"][0]
    note = CASE_NOTES[filename]
    label, css_class = CLASS_META[state["classification"]]
    hero = next(
        player
        for player in request["players"]
        if player["position"] == request["current_player_position"]
    )
    max_bet = max(player["bet"] for player in request["players"])
    call_cost = max(0, max_bet - hero["bet"])
    pot_odds = "—"
    if call_cost:
        pot_odds = "{:.2f}%".format(100.0 * call_cost / (request["current_pot"] + call_cost))
    board = " ".join(card_label(card) for card in request.get("community_cards", [])) or "Preflop"
    hero_cards = " ".join(card_label(card) for card in hero["hole_cards"])
    action = response["action"]
    if response.get("raise_size"):
        action += " to %s" % response["raise_size"]
    action += " · {:.1f}%".format(100.0 * response["action_probability"])

    image_path = safe_image_path(images_dir, filename)
    encoded = base64.b64encode(image_path.read_bytes()).decode("ascii")
    return """
<article class="case {css_class}" data-kind="{css_class}">
  <header>
    <div><span class="num">#{index:02d}</span><div><h2>{title}</h2><p>{filename}</p></div></div>
    <span class="tag">{label}</span>
  </header>
  <div class="facts">
    <div><b>Hero</b><span>{position} · {hero_cards}</span></div>
    <div><b>街道 / 公共牌</b><span>{stage} · {board}</span></div>
    <div><b>底池 / 当前投入</b><span>pot {pot} · Hero {hero_bet} · max {max_bet}</span></div>
    <div><b>面对补注 / 即时赔率</b><span>{call_cost} · {pot_odds}</span></div>
    <div><b>原始策略</b><span class="decision">{action}</span></div>
    <div><b>样本次数</b><span>{occurrences}</span></div>
  </div>
  <div class="verdict"><b>判断</b><span>{reason}</span></div>
  <div class="next"><b>算法侧复核</b><span>{next_step}</span></div>
  <div class="body">
    <figure><img loading="lazy" src="data:image/jpeg;base64,{encoded}" alt="{filename}"><figcaption>原始 Hero 截图 · 页面内嵌</figcaption></figure>
    <section class="evidence">
      <details open><summary>action_history</summary>{history}</details>
      <details><summary>完整决策请求</summary><pre>{request_json}</pre></details>
      <details><summary>原始决策响应</summary><pre>{response_json}</pre></details>
    </section>
  </div>
</article>""".format(
        css_class=css_class,
        index=index,
        title=esc(note["title"]),
        filename=esc(filename),
        label=esc(label),
        position=esc(hero["position"]),
        hero_cards=esc(hero_cards),
        stage=esc(request["game_stage"]),
        board=esc(board),
        pot=esc(request["current_pot"]),
        hero_bet=esc(hero["bet"]),
        max_bet=esc(max_bet),
        call_cost=esc(call_cost),
        pot_odds=esc(pot_odds),
        action=esc(action),
        occurrences=esc(state["occurrences"]),
        reason=esc(note["reason"]),
        next_step=esc(note["next"]),
        encoded=encoded,
        history=render_history(request.get("action_history")),
        request_json=pretty_json(request),
        response_json=pretty_json(response),
    )


def main():
    args = parse_args()
    audit = load_json(args.audit)
    requests = {item["file"]: item["request"] for item in load_json(args.requests)}
    replay = {item["file"]: item for item in load_json_lines(args.replay)}
    images_dir = Path(args.images_dir)

    flagged = [
        state
        for state in audit["states"]
        if state["classification"] in CLASS_META
    ]
    order = {"strict": 0, "risk": 1, "solver": 2}
    flagged.sort(
        key=lambda state: (
            order[CLASS_META[state["classification"]][1]], state["files"][0]
        )
    )
    if len(flagged) != 8:
        raise ValueError("expected 8 flagged states, got %d" % len(flagged))

    cases = []
    for index, state in enumerate(flagged, 1):
        filename = state["files"][0]
        if filename not in CASE_NOTES:
            raise KeyError("missing Chinese note for %s" % filename)
        response = replay[filename].get("accepted_response")
        if response is None:
            raise ValueError("missing accepted response for %s" % filename)
        cases.append(render_case(index, state, requests[filename], response, images_dir))

    template = """<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>2026-10-03 Hero 策略问题复核</title>
<style>
:root{{--bg:#08111f;--panel:#111d30;--panel2:#0c1728;--line:#293b57;--text:#eef5ff;--muted:#9db0ca;--strict:#ff657a;--risk:#ffb25c;--solver:#7ca9ff;--ok:#66d6a7}}*{{box-sizing:border-box}}body{{margin:0;background:linear-gradient(180deg,#07101d,#0a1423 45%,#08111f);color:var(--text);font:14px/1.6 system-ui,-apple-system,"Segoe UI",sans-serif}}main{{max-width:1540px;margin:auto;padding:28px}}h1{{font-size:30px;margin:0 0 8px}}.lead{{max-width:1120px;color:var(--muted);margin:0}}.summary{{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin:22px 0}}.metric,.notice,.toolbar,.case{{background:rgba(17,29,48,.96);border:1px solid var(--line);border-radius:14px}}.metric{{padding:15px}}.metric strong{{display:block;font-size:28px}}.metric span{{color:var(--muted)}}.metric.strict strong{{color:var(--strict)}}.metric.risk strong{{color:var(--risk)}}.metric.solver strong{{color:var(--solver)}}.notice{{padding:14px 16px;border-left:4px solid var(--solver)}}.toolbar{{position:sticky;top:8px;z-index:5;padding:10px;display:flex;gap:8px;margin:18px 0;backdrop-filter:blur(10px)}}button{{border:1px solid var(--line);background:#192944;color:var(--text);padding:8px 13px;border-radius:8px;cursor:pointer;white-space:nowrap}}button.active{{background:#315f9e;border-color:#6899dc}}.case{{margin:16px 0;overflow:hidden}}.case>header{{padding:14px 18px;display:flex;justify-content:space-between;gap:16px;align-items:center;border-bottom:1px solid var(--line)}}.case>header>div{{display:flex;gap:12px;align-items:center}}h2{{font-size:18px;margin:0}}header p{{margin:2px 0 0;color:var(--muted);font-size:12px}}.num{{font-size:17px;font-weight:800;color:var(--solver)}}.tag{{padding:5px 10px;border-radius:999px;font-weight:700;white-space:nowrap}}.strict .tag{{color:#ffd7de;background:#4a1d2b;border:1px solid #813247}}.risk .tag{{color:#ffe4c1;background:#49301c;border:1px solid #7f5328}}.solver .tag{{color:#d7e5ff;background:#1c3153;border:1px solid #345f9d}}.facts{{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px;padding:14px 18px;background:var(--panel2)}}.facts div{{display:grid;gap:2px}}.facts b,.verdict b,.next b{{color:var(--muted);font-size:12px}}.decision{{font-weight:800}}.verdict,.next{{display:grid;grid-template-columns:100px 1fr;gap:10px;padding:12px 18px;border-top:1px solid var(--line)}}.verdict{{background:rgba(255,178,92,.055)}}.next{{background:rgba(124,169,255,.055)}}.body{{display:grid;grid-template-columns:minmax(360px,48%) 1fr;gap:16px;padding:16px}}figure{{margin:0;position:sticky;top:74px;align-self:start}}img{{display:block;width:100%;max-height:790px;object-fit:contain;background:#03070e;border-radius:10px}}figcaption{{color:var(--muted);font-size:12px;margin-top:5px}}.evidence{{min-width:0}}details{{background:var(--panel2);border:1px solid var(--line);border-radius:9px;margin-bottom:12px;overflow:hidden}}summary{{padding:10px 12px;cursor:pointer;font-weight:700;color:#c2d7ff}}pre{{margin:0;padding:12px;border-top:1px solid var(--line);overflow:auto;max-height:520px;color:#d7e4f8;font:12px/1.45 ui-monospace,SFMono-Regular,Menlo,monospace;white-space:pre-wrap;word-break:break-word}}table{{width:100%;border-collapse:collapse}}th,td{{text-align:left;padding:8px 10px;border-top:1px solid var(--line)}}th{{color:var(--muted);font-size:12px}}.empty{{padding:0 12px;color:var(--muted)}}.hidden{{display:none}}footer{{color:var(--muted);padding:18px 2px 4px}}@media(max-width:900px){{main{{padding:14px}}.summary,.facts,.body{{grid-template-columns:1fr}}.toolbar{{overflow:auto}}figure{{position:static}}.verdict,.next{{grid-template-columns:1fr}}}}
</style></head><body><main>
<h1>2026-10-03 Hero 策略问题复核</h1>
<p class="lead">128 次有效决策归并为 84 个唯一状态后，筛出 8 个需要处理或重点复核的局面。页面展示原始截图、完整请求和决策算法原始响应；未在 ddbaby 侧归一化或改写动作。</p>
<div class="summary"><div class="metric"><strong>8</strong><span>问题 / 重点复核状态</span></div><div class="metric strict"><strong>2</strong><span>可严格判错的 free-fold</span></div><div class="metric risk"><strong>4</strong><span>高风险策略异常</span></div><div class="metric solver"><strong>2</strong><span>需同配置 Solver 复核</span></div></div>
<div class="notice"><b>判定边界：</b>只有 2 条 free-fold 无需范围假设即可定错；高风险项不能在缺少范围、rake、完整策略树和各动作 EV 时直接宣称数学错误；solver 敏感项保留为极化策略复核样本。</div>
<nav class="toolbar"><button class="active" data-filter="all">全部 8</button><button data-filter="strict">明确错误 2</button><button data-filter="risk">高风险 4</button><button data-filter="solver">Solver 复核 2</button></nav>
<section id="cases">{cases}</section>
<footer>生成依据：策略审计提交 4b360d1（CR LGTM）。页面为自包含 HTML，8 张图片均内嵌；无生产代码变更。</footer>
</main><script>document.querySelectorAll('[data-filter]').forEach(function(button){{button.addEventListener('click',function(){{document.querySelectorAll('[data-filter]').forEach(function(item){{item.classList.remove('active')}});button.classList.add('active');var filter=button.dataset.filter;document.querySelectorAll('.case').forEach(function(card){{card.classList.toggle('hidden',filter!=='all'&&card.dataset.kind!==filter)}})}})}});</script></body></html>"""
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(template.format(cases="\n".join(cases)), encoding="utf-8")


if __name__ == "__main__":
    main()
