export type Overview = {
  conns: number;
  in_pps: number;
  out_pps: number;
  in_bps: number;
  out_bps: number;
  error_rate: number;
  errors: number;
  ring_used: number;
  ring_cap: number;
  ring_ratio: number;
  reactors: number;
  reactor_load: number[];
  collected_at: string;
};

export type Upstream = {
  id: string;
  host: string;
  port: number;
  weight: number;
  state: string;
  reason?: string;
  active: number;
  idle: number;
  queued: number;
  fails: number;
  in_bytes: number;
  out_bytes: number;
};

export type Node = {
  id: string;
  kind: string;
  label: string;
  status: string;
  reason?: string;
  stats?: Record<string, unknown>;
};

export type Edge = {
  id: string;
  from: string;
  to: string;
  pps: number;
  bps: number;
  status: string;
};

export type Topology = { nodes: Node[]; edges: Edge[]; collected_at: string };

export type Snapshot = {
  overview: Overview;
  topology: Topology;
  upstreams: Upstream[];
};

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: { Accept: "application/json" } });
  if (!res.ok) {
    let msg = res.statusText;
    try {
      const body = await res.json();
      msg = body?.error?.message || msg;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  return res.json();
}

export function fetchOverview() {
  return getJSON<Overview>("/api/v1/overview");
}
export function fetchTopology() {
  return getJSON<Topology>("/api/v1/topology");
}
export function fetchUpstreams() {
  return getJSON<{ upstreams: Upstream[] }>("/api/v1/upstreams").then((x) => x.upstreams);
}
export async function fetchSnapshot(): Promise<Snapshot> {
  const [overview, topology, upstreams] = await Promise.all([
    fetchOverview(),
    fetchTopology(),
    fetchUpstreams(),
  ]);
  return { overview, topology, upstreams };
}

export async function ejectUpstream(id: string, reason: string) {
  const res = await fetch(`/api/v1/upstreams/${encodeURIComponent(id)}/eject`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body?.error?.message || "摘除失败");
  }
}

export async function restoreUpstream(id: string) {
  const res = await fetch(`/api/v1/upstreams/${encodeURIComponent(id)}/restore`, {
    method: "POST",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body?.error?.message || "恢复失败");
  }
}

export function formatBps(bps: number): { value: string; unit: string } {
  const abs = Math.abs(bps);
  if (abs >= 1e9) return { value: (bps / 1e9).toFixed(2), unit: "Gb/s" };
  if (abs >= 1e6) return { value: (bps / 1e6).toFixed(2), unit: "Mb/s" };
  if (abs >= 1e3) return { value: (bps / 1e3).toFixed(1), unit: "Kb/s" };
  return { value: bps.toFixed(0), unit: "b/s" };
}
