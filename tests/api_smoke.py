#!/usr/bin/env python3
"""Offline/mock smoke: talks to the local compose demo, no metered APIs."""
import json
import os
import urllib.request

BASE = os.environ.get("SMOKE_BASE", "http://127.0.0.1:31881")


def get(path):
    with urllib.request.urlopen(BASE + path, timeout=5) as r:
        return r.status, json.loads(r.read().decode())


def main():
    st, health = get("/healthz")
    assert st == 200 and health.get("status") == "ok", health
    st, ov = get("/api/v1/overview")
    assert st == 200 and "conns" in ov, ov
    st, routes = get("/api/v1/routes")
    assert st == 200 and routes.get("routes"), routes
    st, ups = get("/api/v1/upstreams")
    assert st == 200 and len(ups.get("upstreams", [])) >= 3, ups
    st, topo = get("/api/v1/topology")
    assert st == 200 and topo.get("nodes") and topo.get("edges"), topo
    print("api_smoke OK")


if __name__ == "__main__":
    main()
