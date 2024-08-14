#! /usr/bin/env python3

import os
import platform
import subprocess
import sys
from pathlib import Path

ref = os.getenv("GITHUB_REF")  # eg. 'refs/tags/v0.0.3-test'
if ref is None:
    version = "dev"
else:
    version = ref.split("/")[-1]


arch_to_zig_target = {
    "amd64": "x86_64",
    "arm64": "aarch64",
}

goos_to_zig_target = {
    "linux": "linux",
    "darwin": "macos",
}


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

    for prog in ["engine", "iac", "klotho"]:
        prog_env = env.copy()
        if prog == "iac" or prog == "klotho":
            # IaC rendering requires cgo for treesitter parsing of the pulumi ts templates
            # both 'iac' and 'klotho' do iac rendering
            prog_env["CGO_ENABLED"] = "1"

            zig_arch = arch_to_zig_target[arch]
            zig_os = goos_to_zig_target[goos]
            if goos != sys.platform:
                suffix = ""
                # Add -musl suffix for linux builds to support being run in alpine
                # containers and similar environments
                if goos == "linux":
                    suffix = "-musl"
                prog_env["CC"] = f"zig cc -target {zig_arch}-{zig_os}{suffix}"
                prog_env["CXX"] = f"zig c++ -target {zig_arch}-{zig_os}{suffix}"
                print(f"Using zig to build for {goos}/{arch} with {prog_env['CC']}")

        cmd = [*args, f"-o={out_dir}/{prog}_{goos}_{arch}", f"./cmd/{prog}"]
        print(f"running {cmd}")
        proc = subprocess.run(
            cmd,
            stdout=sys.stdout,
            stderr=sys.stderr,
            env=prog_env,
        )
        try:
            proc.check_returncode()
        except subprocess.CalledProcessError as e:
            raise Exception(f"Failed to build for {prog} {goos}/{arch}") from e
        print(f"Successfuly built {prog} for {goos}/{arch}")


targets = [
    ("linux", "amd64"),
    ("darwin", "amd64"),
    ("darwin", "arm64"),
]

if __name__ == "__main__":
    print(f"Building ref {ref} = version {version}")
    print(f"on {sys.platform} {platform.machine()}")
    for goos, arch in targets:
        release(goos, arch, "dist")
