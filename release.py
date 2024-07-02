#! /usr/bin/env python3

import os
import subprocess
import sys
from pathlib import Path

ref = os.getenv("GITHUB_REF")
if ref is None:
    version = "dev"
else:
    version = ref.split("/")[-1]


def release(goos, arch, out_dir):
    env = os.environ.copy()
    env["GOOS"] = goos
    env["GOARCH"] = arch

    args = [
        "go",
        "build",
        "-ldflags",
        f"-X 'main.version={version}' -s -w",
    ]

    Path(out_dir).mkdir(parents=True, exist_ok=True)

    for prog in ["engine", "iac", "k2"]:
        prog_env = env.copy()
        if prog == "iac" or prog == "k2":
            # IaC rendering requires cgo for treesitter parsing of the pulumi ts templates
            # both 'iac' and 'k2' do iac rendering
            prog_env["CGO_ENABLED"] = "1"

            if goos == "linux":
                prog_env["CC"] = "zig cc -target x86_64-linux-musl"
                prog_env["CXX"] = "zig c++ -target x86_64-linux-musl"

        cmd = [*args, f"-o={out_dir}/{prog}_{goos}_{arch}", f"./cmd/{prog}"]
        print(f"running {cmd}")
        proc = subprocess.run(
            cmd,
            stdout=sys.stdout,
            stderr=sys.stderr,
            env=env,
        )
        try:
            proc.check_returncode()
        except subprocess.CalledProcessError as e:
            raise Exception(f"Failed to build for {prog} {goos}/{arch}") from e


targets = [
    ("linux", "amd64"),
    ("darwin", "amd64"),
    ("darwin", "arm64"),
]

if __name__ == "__main__":
    for goos, arch in targets:
        release(goos, arch, "dist")
