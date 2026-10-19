#!/usr/bin/env python3
"""Start the NovelStudio API and web development server."""

from __future__ import annotations

import argparse
import os
import shutil
import signal
import subprocess
import sys
import time
from pathlib import Path


ROOT = Path(__file__).resolve().parent
WEB_DIR = ROOT / "web"


def executable(name: str) -> str:
    """Return an executable path or stop with a useful error."""
    path = shutil.which(name)
    if path:
        return path
    print(f"错误：未找到 {name}，请先安装并将其加入 PATH。", file=sys.stderr)
    raise SystemExit(1)


def run_install(npm: str) -> None:
    print("首次安装前端依赖……")
    subprocess.run([npm, "install"], cwd=WEB_DIR, check=True)


def start_process(command: list[str], cwd: Path, label: str) -> subprocess.Popen[bytes]:
    print(f"启动 {label}: {' '.join(command)}")
    kwargs: dict[str, object] = {"cwd": cwd}
    if os.name == "nt":
        kwargs["creationflags"] = subprocess.CREATE_NEW_PROCESS_GROUP
    else:
        kwargs["start_new_session"] = True
    return subprocess.Popen(command, **kwargs)


def stop_process(process: subprocess.Popen[bytes]) -> None:
    if process.poll() is not None:
        return
    try:
        if os.name == "nt":
            process.send_signal(signal.CTRL_BREAK_EVENT)
        else:
            os.killpg(process.pid, signal.SIGTERM)
        process.wait(timeout=5)
    except (OSError, subprocess.TimeoutExpired):
        process.kill()
        process.wait()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="启动 NovelStudio 开发环境")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--api-only", action="store_true", help="只启动 Go API")
    mode.add_argument("--web-only", action="store_true", help="只启动 Web 前端")
    parser.add_argument(
        "--install",
        action="store_true",
        help="启动前执行 npm install（缺少 node_modules 时也会自动执行）",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    processes: list[tuple[str, subprocess.Popen[bytes]]] = []

    try:
        if not args.web_only:
            go = executable("go")
            processes.append(("API", start_process([go, "run", "./cmd/api"], ROOT, "Go API")))

        if not args.api_only:
            npm = executable("npm.cmd" if os.name == "nt" else "npm")
            if args.install or not (WEB_DIR / "node_modules").is_dir():
                run_install(npm)
            processes.append(("Web", start_process([npm, "run", "dev"], WEB_DIR, "Vite Web")))

        print("\nNovelStudio 正在运行：")
        if not args.web_only:
            print("  API: http://localhost:8081")
        if not args.api_only:
            print("  Web: http://localhost:5173")
        print("按 Ctrl+C 停止全部服务。\n")

        while True:
            for label, process in processes:
                code = process.poll()
                if code is not None:
                    print(f"{label} 已退出（退出码 {code}）。", file=sys.stderr)
                    return code or 1
            time.sleep(0.5)
    except KeyboardInterrupt:
        print("\n正在停止服务……")
        return 0
    except subprocess.CalledProcessError as error:
        print(f"命令执行失败（退出码 {error.returncode}）。", file=sys.stderr)
        return error.returncode
    finally:
        for _, process in reversed(processes):
            stop_process(process)


if __name__ == "__main__":
    raise SystemExit(main())
