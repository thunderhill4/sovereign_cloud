import React from 'react';
import {AbsoluteFill, interpolate, useCurrentFrame} from 'remotion';
import {C, FONT, MONO, sec} from '../theme';
import {B, Captions, Card, Chip, Enter, Heading} from '../components/ui';

// A chat pod spawns every SPAWN seconds from t=9, lives LIFE seconds.
const SPAWN = 2.6;
const LIFE = 6.5;
const NAMES = ['a7k2', 'm3qx', 'p9d1', 'z4tw', 'h8ve', 'c2nr', 'w6jb', 'f1ls'];

export const S06Runs: React.FC = () => {
  const frame = useCurrentFrame();
  const t = frame / 30;
  const beat = 0.5 + 0.5 * Math.sin(frame / 12);

  const pods = NAMES.map((n, i) => {
    const born = 9 + i * SPAWN;
    const age = t - born;
    if (age < 0 || age > LIFE) return null;
    const inP = Math.min(1, age / 0.5);
    const outP = Math.max(0, (age - (LIFE - 0.6)) / 0.6);
    const slot = i % 3;
    return {n, slot, op: inP * (1 - outP), age, born};
  });

  return (
    <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
      <Heading kicker="Chapter 5 · Two kinds of pods" title="One door that stays open, many workers that come and go" />

      <Enter delay={1} scale style={{left: 120, top: 270}}>
        <Card accent={C.cluster2} glow={beat * 0.6} style={{width: 640, height: 520, padding: '32px 36px'}}>
          <div style={{fontSize: 20, letterSpacing: 3, color: C.cluster2, fontWeight: 800}}>ALWAYS ON · SERVING</div>
          <div style={{fontSize: 36, fontWeight: 800, marginTop: 10}}>🚪 The front door</div>
          <div style={{fontFamily: MONO, fontSize: 22, color: C.dim, marginTop: 14}}>cluster2-agent-web-endpoint-server</div>
          <div style={{fontSize: 24, color: C.dim, marginTop: 20, lineHeight: 1.5}}>
            Created because the agent lists the <B c={C.cluster2}>web-endpoint</B> skill. It holds a fixed MetalLB address and waits for chats.
          </div>
          <Chip color={C.cluster2}>172.18.255.213:8080</Chip>
          <Chip color={C.cluster2}>long-lived Deployment</Chip>
        </Card>
      </Enter>

      <svg width={1920} height={1080} style={{position: 'absolute'}}>
        {[0, 1, 2].map((s) => (
          <line key={s} x1={760} y1={530} x2={1060} y2={330 + s * 190} stroke="#2c3a5c" strokeWidth={3} strokeDasharray="8 10" opacity={interpolate(frame, [sec(8), sec(9)], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'})} />
        ))}
        {pods.map((p) => {
          if (!p || p.age > 0.9) return null;
          const q = p.age / 0.9;
          return <circle key={p.n} cx={760 + 300 * q} cy={530 + (330 + p.slot * 190 - 530) * q} r={11} fill={C.target} />;
        })}
      </svg>

      <Enter delay={6} dy={20} style={{left: 1060, top: 220}}>
        <div style={{fontSize: 20, letterSpacing: 3, color: C.target, fontWeight: 800}}>ONE PER CHAT · TASK RUNS</div>
      </Enter>
      {pods.map((p) =>
        p ? (
          <div key={p.n} style={{position: 'absolute', left: 1060 + (p.slot % 2) * 30, top: 270 + p.slot * 190, opacity: p.op, transform: `translateX(${(1 - Math.min(1, p.age / 0.5)) * 40}px)`}}>
            <Card accent={C.target} glow={0.5} style={{width: 720, height: 150, padding: '20px 28px', display: 'flex', alignItems: 'center', gap: 22}}>
              <span style={{fontSize: 46}}>📦</span>
              <div>
                <div style={{fontSize: 27, fontWeight: 800}}>
                  AgentRun <span style={{fontFamily: MONO, color: C.target}}>chat-{p.n}</span>
                </div>
                <div style={{fontSize: 20, color: C.dim, marginTop: 6, fontFamily: MONO}}>identity: sympozium-run-chat-{p.n}</div>
              </div>
              <div style={{marginLeft: 'auto', fontFamily: MONO, fontSize: 22, color: p.age > LIFE - 1 ? C.faint : C.target}}>
                {p.age > LIFE - 1 ? 'done ✓' : 'working…'}
              </div>
            </Card>
          </div>
        ) : null,
      )}

      <Captions
        lines={[
          {at: 1, text: <>Each agent has a <B c={C.cluster2}>front door</B>: a long-running pod with a fixed address that accepts chats.</>},
          {at: 9, text: <>Every chat that arrives gets its own short-lived <B c={C.target}>AgentRun</B> pod to do the work.</>},
          {at: 16, text: 'Each run also gets its own identity, a ServiceAccount created for it and deleted with it.'},
          {at: 23, text: 'When the answer is sent, the pod finishes. Nothing is left running between questions.'},
        ]}
      />
    </AbsoluteFill>
  );
};
