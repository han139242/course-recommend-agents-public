from __future__ import annotations

from pathlib import Path

import matplotlib.pyplot as plt
from matplotlib.patches import Ellipse, FancyArrowPatch, FancyBboxPatch


ROOT = Path(__file__).resolve().parent
FIGSIZE = (16, 9)
DPI = 100

BG = "#F6FAFE"
TEXT = "#1C2A3D"
MUTED = "#465B75"
NAVY = "#486383"
LINE = "#7590B2"
BLUE = "#E7F0FB"
MINT = "#E8F5EF"
WARM = "#FCEEE1"
PANEL = "#FCFEFF"
SHADOW = "#B8C8DB"
ACCENT_BLUE = "#88A8CC"
ACCENT_MINT = "#90C6B0"
ACCENT_WARM = "#D89A77"


def canvas():
    fig = plt.figure(figsize=FIGSIZE, dpi=DPI)
    ax = plt.axes([0, 0, 1, 1])
    ax.set_xlim(0, 1600)
    ax.set_ylim(0, 900)
    ax.axis("off")
    fig.patch.set_facecolor(BG)
    ax.set_facecolor(BG)
    return fig, ax


def box(ax, x, y, w, h, fill, edge=NAVY, lw=2.2, radius=24, shadow=True, z=2):
    if shadow:
        ax.add_patch(
            FancyBboxPatch(
                (x + 10, y - 10),
                w,
                h,
                boxstyle=f"round,pad=0.012,rounding_size={radius}",
                linewidth=0,
                facecolor=SHADOW,
                alpha=0.18,
                zorder=z - 1,
            )
        )
    patch = FancyBboxPatch(
        (x, y),
        w,
        h,
        boxstyle=f"round,pad=0.012,rounding_size={radius}",
        linewidth=lw,
        edgecolor=edge,
        facecolor=fill,
        zorder=z,
    )
    ax.add_patch(patch)
    return patch


def txt(ax, x, y, s, *, size=16, weight="bold", color=TEXT, ha="center", va="center", z=20):
    ax.text(
        x,
        y,
        s,
        fontsize=size,
        fontweight=weight,
        color=color,
        family="DejaVu Sans",
        ha=ha,
        va=va,
        zorder=z,
    )


def node(ax, x, y, w, h, head, sub=None, fill=PANEL, edge=LINE):
    box(ax, x, y, w, h, fill, edge=edge, lw=1.8, radius=18, shadow=False, z=5)
    head_size = 15
    if len(head.replace("\n", "")) > 24:
        head_size = 13
    if len(head.replace("\n", "")) > 34:
        head_size = 12
    txt(ax, x + w / 2, y + h - 22, head, size=head_size)
    if sub:
        lines = sub.split("\n")
        for i, line in enumerate(lines):
            txt(ax, x + w / 2, y + h - 47 - i * 16, line, size=11.5, weight="normal", color=MUTED)


def arrow(ax, start, end, *, color=NAVY, lw=3, ms=18, curve=0.0, ls="-", z=10):
    ax.add_patch(
        FancyArrowPatch(
            start,
            end,
            connectionstyle=f"arc3,rad={curve}",
            arrowstyle="-|>",
            mutation_scale=ms,
            linewidth=lw,
            linestyle=ls,
            color=color,
            zorder=z,
        )
    )


def loop_hub(ax, cx, cy, w, h, top_text, bottom_text):
    ax.add_patch(Ellipse((cx, cy), w, h, facecolor="#EAF2FB", edgecolor=NAVY, linewidth=2.6, zorder=3))
    ax.add_patch(Ellipse((cx, cy), w - 38, h - 38, facecolor="#F8FBFF", edgecolor=ACCENT_BLUE, linewidth=1.6, zorder=4))
    arrow(ax, (cx - 110, cy + 95), (cx + 110, cy + 95), color=ACCENT_BLUE, lw=4.8, ms=20, curve=0.23)
    arrow(ax, (cx + 120, cy - 95), (cx - 120, cy - 95), color=ACCENT_WARM, lw=4.8, ms=20, curve=0.23)
    if top_text:
        txt(ax, cx, cy + h / 2 - 72, top_text, size=13, color=MUTED)
    if bottom_text:
        txt(ax, cx, cy - h / 2 + 34, bottom_text, size=13, color=MUTED)


def learning_diagnosis(ax):
    txt(ax, 800, 862, "[LEARNING DIAGNOSIS FRAMEWORK]", size=31)

    box(ax, 70, 285, 340, 435, BLUE)
    txt(ax, 240, 685, "[TRACE LAYER]", size=22)
    node(ax, 105, 575, 270, 88, "ANSWER LOGS", "accuracy\nwrong answers\nattempt order")
    node(ax, 105, 470, 270, 88, "PROGRESS SIGNALS", "chapter status\npath completion\nstage progress")
    node(ax, 105, 365, 270, 88, "MASTERY VIEW", "mastery level\nweak area map\nsummary cards")
    node(ax, 105, 305, 270, 42, "PROFILE EVIDENCE READY")

    loop_hub(ax, 790, 480, 600, 430, "", "PASSIVE RESPONSE TO ACTIVE DIAGNOSIS")
    txt(ax, 790, 690, "[DIAGNOSIS HUB]", size=24)
    node(ax, 610, 500, 170, 108, "PROFILE UPDATE", "records + progress\n+ mastery")
    node(ax, 800, 490, 200, 132, "RULE ENGINE", "threshold rules\nstage checks\nmanual policy")
    node(ax, 675, 332, 235, 108, "CURRENT OUTPUT", "display-oriented hints\nassisted explanation")
    arrow(ax, (780, 554), (800, 554), color=LINE, lw=2.8, ms=16)
    arrow(ax, (900, 485), (840, 440), color=LINE, lw=2.8, ms=16)

    box(ax, 1135, 500, 370, 220, MINT)
    txt(ax, 1320, 685, "[CURRENT BASELINE]", size=22)
    node(ax, 1165, 612, 310, 44, "ANSWER / PROGRESS / MASTERY LOGGING")
    node(ax, 1165, 553, 310, 44, "PROFILE DASHBOARD AND GROWTH VIEW")
    node(ax, 1165, 494, 310, 44, "RECOMMENDATION LINKED WITH GRAPH VIEW")

    box(ax, 1105, 155, 410, 265, WARM)
    txt(ax, 1310, 385, "[NEXT UPGRADE]", size=22)
    node(ax, 1135, 300, 170, 72, "CONTINUOUS\nSENSING", "richer behavior traces", fill="#FFF8F2")
    node(ax, 1310, 300, 170, 72, "AUTO\nDIAGNOSIS", "weak-point detection", fill="#FFF8F2")
    node(ax, 1135, 208, 170, 72, "REPAIR\nPATH", "targeted recovery route", fill="#FFF8F2")
    node(ax, 1310, 208, 170, 72, "EXPLAINABLE\nREPORT", "rationale + evidence", fill="#FFF8F2")

    box(ax, 150, 60, 920, 88, PANEL, edge=LINE, lw=2.0, radius=24, shadow=False)
    txt(ax, 610, 118, "[EVOLUTION LOOP]", size=20)
    txt(ax, 230, 84, "DATA LOGGING", size=14)
    txt(ax, 465, 84, "STAGE ANALYSIS", size=14)
    txt(ax, 705, 84, "DIAGNOSIS OUTPUT", size=14)
    txt(ax, 940, 84, "PATH REPAIR", size=14)
    arrow(ax, (295, 84), (380, 84), color=LINE, lw=2.8, ms=16)
    arrow(ax, (545, 84), (630, 84), color=LINE, lw=2.8, ms=16)
    arrow(ax, (805, 84), (885, 84), color=LINE, lw=2.8, ms=16)

    arrow(ax, (410, 505), (495, 505), color=NAVY, lw=4.1, ms=18)
    arrow(ax, (1095, 585), (1135, 585), color=ACCENT_MINT, lw=3.3, ms=18)
    arrow(ax, (1095, 290), (1135, 290), color=ACCENT_WARM, lw=3.3, ms=18)
    arrow(ax, (1320, 500), (1320, 425), color=LINE, lw=2.0, ms=16, ls="--")
    txt(ax, 1352, 452, "upgrade path", size=11, weight="normal", color=MUTED, ha="left")


def interaction_optimization(ax):
    txt(ax, 800, 862, "[INTERACTION OPTIMIZATION FRAMEWORK]", size=31)

    box(ax, 70, 510, 340, 225, BLUE)
    txt(ax, 240, 700, "[CORE FLOW]", size=22)
    node(ax, 100, 625, 280, 50, "GRAPH BROWSING")
    node(ax, 100, 560, 280, 50, "PATH RECOMMENDATION")
    node(ax, 100, 495, 280, 50, "AI-ASSISTED LEARNING")

    loop_hub(ax, 790, 500, 590, 430, "", "STABILITY AND USABILITY")
    txt(ax, 790, 708, "[INTERACTION HUB]", size=24)
    node(ax, 610, 565, 165, 98, "SESSION\nCONTEXT", "current node\npath state\npage context")
    node(ax, 805, 565, 165, 98, "UI STATE\nSYNC", "graph\nside panel\nchat panel")
    node(ax, 655, 430, 270, 122, "PRIORITY RISKS", "conversation flow\nerror hints\nweak-network fallback\nloading feedback")
    arrow(ax, (775, 613), (805, 613), color=LINE, lw=2.8, ms=16)
    arrow(ax, (888, 565), (878, 552), color=LINE, lw=2.8, ms=16)

    box(ax, 1135, 505, 370, 230, MINT)
    txt(ax, 1320, 700, "[PRESENTATION LAYER]", size=22)
    node(ax, 1165, 625, 310, 50, "GRAPH / DETAIL / AI CHAT LINKAGE")
    node(ax, 1165, 560, 310, 50, "LOWER INFORMATION DENSITY")
    node(ax, 1165, 495, 310, 50, "CLEARER FOCUS AND VIEW TRANSITION")

    box(ax, 1135, 165, 370, 255, WARM)
    txt(ax, 1320, 385, "[RELIABILITY LAYER]", size=22)
    node(ax, 1165, 315, 310, 52, "MODEL-FLUCTUATION FALLBACK")
    node(ax, 1165, 245, 310, 52, "WEAK-NETWORK RETRY AND FEEDBACK")
    node(ax, 1165, 175, 310, 52, "MOBILE AND MULTI-TERMINAL ADAPTATION")

    box(ax, 70, 185, 340, 225, PANEL)
    txt(ax, 240, 375, "[KNOWN GAPS]", size=22)
    txt(ax, 105, 320, "• conversation continuity needs refinement", size=13, weight="normal", ha="left")
    txt(ax, 105, 278, "• error prompts are not fully unified", size=13, weight="normal", ha="left")
    txt(ax, 105, 236, "• weak-network experience still fluctuates", size=13, weight="normal", ha="left")

    box(ax, 120, 55, 1360, 82, PANEL, edge=LINE, lw=2.0, radius=24, shadow=False)
    txt(ax, 800, 112, "[OPTIMIZATION LOOP]", size=20)
    txt(ax, 225, 80, "PROBLEM FINDING", size=14)
    txt(ax, 545, 80, "INTERACTION TUNING", size=14)
    txt(ax, 930, 80, "FRONTEND + RELIABILITY", size=14)
    txt(ax, 1270, 80, "EXPERIENCE UPGRADE", size=14)
    arrow(ax, (300, 80), (445, 80), color=LINE, lw=2.8, ms=16)
    arrow(ax, (650, 80), (800, 80), color=LINE, lw=2.8, ms=16)
    arrow(ax, (1060, 80), (1175, 80), color=LINE, lw=2.8, ms=16)

    arrow(ax, (410, 620), (505, 620), color=NAVY, lw=4.1, ms=18)
    arrow(ax, (1095, 620), (1135, 620), color=ACCENT_MINT, lw=3.3, ms=18)
    arrow(ax, (1095, 295), (1135, 295), color=ACCENT_WARM, lw=3.3, ms=18)
    arrow(ax, (410, 300), (545, 470), color=ACCENT_BLUE, lw=2.5, ms=16, ls="--")
    txt(ax, 432, 352, "risk input", size=11, weight="normal", color=MUTED, ha="left")


def render(name: str, draw_fn):
    fig, ax = canvas()
    draw_fn(ax)
    out = ROOT / name
    fig.savefig(out, dpi=DPI, facecolor=BG)
    plt.close(fig)
    print(out)


if __name__ == "__main__":
    render("learning-behavior-diagnosis-framework-v4.png", learning_diagnosis)
    render("interaction-experience-optimization-framework-v4.png", interaction_optimization)
