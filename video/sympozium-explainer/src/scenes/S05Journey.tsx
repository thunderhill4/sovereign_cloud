import React from 'react';
import {AbsoluteFill, Easing, interpolate, useCurrentFrame} from 'remotion';
import {C, FONT, MONO, sec} from '../theme';
import {B, Captions, Card, Code, Line, useEnter} from '../components/ui';

type NodeId = 'you' | 'ui' | 'guard' | 'ep' | 'pod' | 'llm' | 'api';
const N: Record<NodeId, {x: number; y: number; w: number; h: number; icon: string; t: string; d: string}> = {
  you: {x: 70, y: 330, w: 250, h: 150, icon: '🧑‍💻', t: 'You', d: 'dashboard AI tab'},
  ui: {x: 380, y: 330, w: 250, h: 150, icon: '🖥️', t: 'kubeui backend', d: '/api/ai/chat'},
  guard: {x: 690, y: 330, w: 250, h: 150, icon: '🛡️', t: 'Laya guard', d: 'safety check'},
  ep: {x: 1000, y: 330, w: 250, h: 150, icon: '🚪', t: 'Agent endpoint', d: '.213 · /v1/chat'},
  pod: {x: 1320, y: 290, w: 330, h: 230, icon: '📦', t: 'AgentRun pod', d: ''},
  llm: {x: 1110, y: 640, w: 270, h: 150, icon: '🧠', t: 'Ollama', d: 'qwen2.5:7b'},
  api: {x: 1560, y: 640, w: 290, h: 150, icon: '☸️', t: 'Kubernetes API', d: 'real cluster state'},
};
const center = (id: NodeId): [number, number] => [N[id].x + N[id].w / 2, N[id].y + N[id].h / 2];

const EDGES: [NodeId, NodeId][] = [
  ['you', 'ui'],
  ['ui', 'guard'],
  ['guard', 'ep'],
  ['ep', 'pod'],
  ['pod', 'llm'],
  ['pod', 'api'],
];

type Step = {from: number; to: number; path: NodeId[]; color: string; cap: React.ReactNode};
const STEPS: Step[] = [
  {from: 4, to: 11.5, path: ['you', 'ui'], color: C.mesh, cap: <>You type a question in the dashboard's AI tab. It goes to the <B c={C.mesh}>kubeui backend</B>.</>},
  {from: 11.5, to: 19.5, path: ['ui', 'guard'], color: C.cluster2, cap: <>Before any AI runs, the <B c={C.cluster2}>Laya guard</B> scores it: is it destructive? Is it a trick?</>},
  {from: 19.5, to: 27, path: ['guard', 'ep'], color: C.cluster2, cap: <>It is safe, so it is forwarded to the agent's chat address, the same API style as OpenAI.</>},
  {from: 27, to: 35, path: ['ep', 'pod'], color: C.cluster2, cap: <>Sympozium starts an <B c={C.cluster2}>AgentRun</B>: a small pod created just for this one conversation.</>},
  {from: 35, to: 42.5, path: ['pod', 'llm', 'pod'], color: C.model, cap: <>The model reads the question and decides: <B>"I need to look. Call the kubectl tool."</B></>},
  {from: 42.5, to: 50, path: ['pod', 'api', 'pod'], color: C.model, cap: <>The <B c={C.target}>k8s-ops</B> sidecar runs <Code>kubectl get nodes</Code> against the real cluster.</>},
  {from: 50, to: 58, path: ['pod', 'ep', 'guard', 'ui', 'you'], color: C.target, cap: <>The model turns the real output into an answer, streamed back to you word by word.</>},
];

const along = (path: NodeId[], p: number): [number, number] => {
  const pts = path.map(center);
  const lens = pts.slice(1).map((q, i) => Math.hypot(q[0] - pts[i][0], q[1] - pts[i][1]));
  let d = p * lens.reduce((a, b) => a + b, 0);
  for (let i = 0; i < lens.length; i++) {
    if (d <= lens[i] || i === lens.length - 1) {
      const q = lens[i] ? Math.min(1, d / lens[i]) : 0;
      return [pts[i][0] + (pts[i + 1][0] - pts[i][0]) * q, pts[i][1] + (pts[i + 1][1] - pts[i][1]) * q];
    }
    d -= lens[i];
  }
  return pts[pts.length - 1];
};

const Bubble: React.FC<{show: number; x: number; y: number; color: string; w?: number; mono?: boolean; children: React.ReactNode}> = ({show, x, y, color, w, mono, children}) => (
  <div
    style={{
      position: 'absolute',
      left: x,
      top: y,
      width: w,
      opacity: show,
      transform: `translateY(${(1 - show) * 14}px) scale(${0.94 + 0.06 * show})`,
      background: C.card,
      border: `2px solid ${color}`,
      borderRadius: 18,
      padding: '14px 22px',
      fontSize: mono ? 21 : 25,
      fontFamily: mono ? MONO : FONT,
      color,
      lineHeight: 1.4,
      whiteSpace: w ? 'normal' : 'nowrap',
      boxShadow: `0 0 30px ${color}33`,
    }}
  >
    {children}
  </div>
);

export const S05Journey: React.FC = () => {
  const frame = useCurrentFrame();
  const t = frame / 30;
  const stepIdx = STEPS.findIndex((s) => t >= s.from && t < s.to);
  const cur = stepIdx >= 0 ? STEPS[stepIdx] : null;
  const head = useEnter(0.2);

  // window helper: 0→1 in, 1→0 out
  const win = (a: number, b: number) =>
    interpolate(frame, [sec(a), sec(a) + 10, sec(b) - 10, sec(b)], [0, 1, 1, 0], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});

  let packet: [number, number] | null = null;
  let target: NodeId | null = null;
  if (cur) {
    const p = interpolate(frame, [sec(cur.from) + 15, sec(cur.to) - 30], [0, 1], {
      extrapolateLeft: 'clamp',
      extrapolateRight: 'clamp',
      easing: Easing.inOut(Easing.cubic),
    });
    packet = along(cur.path, p);
    target = cur.path[Math.min(cur.path.length - 1, Math.ceil(p * (cur.path.length - 1)))];
  }
  const onEdge = (a: NodeId, b: NodeId) =>
    !!cur && cur.path.some((k, i) => i < cur.path.length - 1 && ((k === a && cur.path[i + 1] === b) || (k === b && cur.path[i + 1] === a)));

  const sidecarLit = t > 42.5 && t < 50;
  const agentLit = t > 35 && t < 42.5;

  return (
    <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
      <div style={{position: 'absolute', left: 120, top: 96, opacity: head}}>
        <div style={{fontSize: 20, letterSpacing: 5, textTransform: 'uppercase', color: C.faint, fontWeight: 700}}>Chapter 4 · Follow one question</div>
        <div style={{fontSize: 54, fontWeight: 800, letterSpacing: -1, marginTop: 8}}>From your keyboard to the cluster and back</div>
      </div>
      {cur && (
        <div style={{position: 'absolute', right: 120, top: 120, fontSize: 24, fontWeight: 900, letterSpacing: 3, color: C.bg0, background: cur.color, borderRadius: 99, padding: '10px 22px'}}>
          STEP {stepIdx + 1} / {STEPS.length}
        </div>
      )}

      <svg width={1920} height={1080} style={{position: 'absolute'}}>
        {EDGES.map(([a, b]) => {
          const [x1, y1] = center(a);
          const [x2, y2] = center(b);
          const lit = onEdge(a, b);
          return (
            <line key={a + b} x1={x1} y1={y1} x2={x2} y2={y2} stroke={lit ? cur!.color : '#2c3a5c'} strokeWidth={lit ? 6 : 3} strokeDasharray={lit ? undefined : '8 10'} opacity={head} />
          );
        })}
        {packet && (
          <>
            <circle cx={packet[0]} cy={packet[1]} r={30 + 6 * Math.sin(frame / 4)} fill={cur!.color} opacity={0.25} />
            <circle cx={packet[0]} cy={packet[1]} r={13} fill={cur!.color} />
          </>
        )}
      </svg>

      {(Object.keys(N) as NodeId[]).map((id, i) => {
        const n = N[id];
        const e = interpolate(frame, [sec(0.6 + i * 0.25), sec(0.6 + i * 0.25) + 12], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});
        const lit = target === id;
        return (
          <div key={id} style={{position: 'absolute', left: n.x, top: n.y, opacity: e, transform: `scale(${0.9 + 0.1 * e})`}}>
            <Card accent={cur?.color} glow={lit ? 1 : 0} style={{width: n.w, height: n.h, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', textAlign: 'center', padding: 14}}>
              <div style={{fontSize: 38}}>{n.icon}</div>
              <div style={{fontSize: 25, fontWeight: 800, marginTop: 4}}>{n.t}</div>
              {n.d && <div style={{fontSize: 19, color: C.dim, marginTop: 4, fontFamily: id === 'ui' ? MONO : FONT}}>{n.d}</div>}
              {id === 'pod' && (
                <div style={{display: 'flex', gap: 12, marginTop: 12}}>
                  <div style={{fontSize: 18, fontFamily: MONO, padding: '8px 12px', borderRadius: 10, border: `2px solid ${agentLit ? C.model : C.border}`, color: agentLit ? C.model : C.dim}}>agent</div>
                  <div style={{fontSize: 18, fontFamily: MONO, padding: '8px 12px', borderRadius: 10, border: `2px solid ${sidecarLit ? C.target : C.border}`, color: sidecarLit ? C.target : C.dim}}>k8s-ops</div>
                </div>
              )}
            </Card>
          </div>
        );
      })}

      {/* step-specific call-outs */}
      <Bubble show={win(4.5, 58)} x={70} y={240} color={C.mesh}>“Are any nodes NotReady?”</Bubble>
      <Bubble show={win(13, 19.5)} x={640} y={510} color={C.cluster2} mono>destructive 0.02 · override 0.01 ✓</Bubble>
      <Bubble show={win(21, 27)} x={900} y={510} color={C.cluster2} mono>POST /v1/chat/completions</Bubble>
      <Bubble show={win(29, 35)} x={1320} y={200} color={C.cluster2} mono>sympozium-run-… (new pod)</Bubble>
      <Bubble show={win(37.5, 42.5)} x={590} y={680} color={C.model} w={500}>tool_call → <span style={{fontFamily: MONO}}>kubectl get nodes</span></Bubble>
      <Bubble show={win(45, 50)} x={1530} y={532} color={C.model} mono w={330}>
        NAME&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;STATUS
        <br />
        cluster2-…&nbsp;&nbsp;&nbsp;&nbsp;Ready
      </Bubble>
      <Bubble show={win(53, 60)} x={70} y={540} color={C.target} w={560}>
        “No. All nodes are <b>Ready</b>, including <span style={{fontFamily: MONO, fontSize: 21}}>cluster2-control-plane</span>.”
      </Bubble>

      <Captions lines={[{at: 0.5, text: 'Let us follow one question all the way through the system.'} as Line, ...STEPS.map((s) => ({at: s.from, text: s.cap}))]} />
    </AbsoluteFill>
  );
};
