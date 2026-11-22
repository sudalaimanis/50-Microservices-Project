#!/usr/bin/env python3
# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
"""Decides which of the 50 images CI must rebuild for this commit.

Rules
  services/<name>/**  -> rebuild that service
  libs/<lang>kit/**   -> rebuild every service written in that language
  services.json       -> rebuild frontend (it ships a copy)
  BUILD_ALL=true, no previous commit, or a new repo -> rebuild everything
Everything else (Helm, docs, workflows) rebuilds nothing; unchanged images are re-tagged.

Writes JSON lists to $GITHUB_OUTPUT: changed, unchanged.
"""
import json
import os
import subprocess

services = json.load(open("services.json"))
names = [s["name"] for s in services]
by_lang = {}
for s in services:
    by_lang.setdefault(s["lang"], []).append(s["name"])

base = os.environ.get("BASE_SHA", "")
build_all = os.environ.get("BUILD_ALL", "false") == "true" or not base or set(base) == {"0"}
changed = set()
if not build_all:
    try:
        files = subprocess.check_output(["git", "diff", "--name-only", base, "HEAD"], text=True).split()
    except subprocess.CalledProcessError:
        files, build_all = [], True
    for f in files:
        parts = f.split("/")
        if parts[0] == "services" and len(parts) > 1 and parts[1] in names:
            changed.add(parts[1])
        elif parts[0] == "libs" and len(parts) > 1:
            lang = parts[1].replace("kit", "")
            changed.update(by_lang.get(lang, []))
        elif f == "services.json":
            changed.add("frontend")
if build_all:
    changed = set(names)

changed_list = [n for n in names if n in changed]
unchanged_list = [n for n in names if n not in changed]
print(f"rebuild {len(changed_list)}: {changed_list}")
print(f"re-tag  {len(unchanged_list)}")
with open(os.environ.get("GITHUB_OUTPUT", "/dev/stdout"), "a") as out:
    out.write(f"changed={json.dumps(changed_list)}\n")
    out.write(f"unchanged={json.dumps(unchanged_list)}\n")
