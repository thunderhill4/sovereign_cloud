import React from 'react';
import {AbsoluteFill, interpolate, useCurrentFrame} from 'remotion';
import {C, FONT, MONO, sec} from '../theme';
import {B, Captions, Card, Heading} from '../components/ui';

type Seg = [cls: 'k' | 's' | 'c', text: string];
// Abridged from 06-sympozium/cluster2-agent.yaml
const YAML: Seg[][] = [
  [['c', '# 06-sympozium/cluster2-agent.yaml']],
  [['k', 'apiVersion: '], ['s', 'sympozium.ai/v1alpha1']],
  [['k', 'kind: '], ['s', 'Agent']],
  [['k', 'metadata:']],
  [['k', '  name: '], ['s', 'cluster2-agent']],
  [['k', 'spec:']],
  [['k', '  agents:']],
  [['k', '    default:']],
  [['k', '      model: '], ['s', 'qwen2.5:7b']],
  [['k', '  skills:']],
  [['k', '    - skillPackRef: '], ['s', 'web-endpoint']],
  [['k', '    - skillPackRef: '], ['s', 'k8s-ops']],
  [['k', '  policyRef: '], ['s', 'sandbox-restricted']],
  [['k', '  memory:']],
  [['k', '    systemPrompt: '], ['s', '|']],
  [['c', '      You operate "cluster2", the management']],
  [['c', '      cluster of a KubeVirt platform ...']],
  [['c', '      Read-only unless the user approves.']],
];
const CLS = {k: '#79c0ff', s: '#a5d6ff', c: '#7d8cab'};

const PARTS = [
  {n: 1, lines: [8], color: C.cluster2, title: 'Model — the brain', d: 'Which language model does the thinking.', at: 9},
  {n: 2, lines: [10, 11], color: C.target, title: 'Skills — the hands', d: 'web-endpoint gives it a chat address. k8s-ops gives it a kubectl tool.', at: 15},
  {n: 3, lines: [12], color: C.policy, title: 'Policy — the rules', d: 'What it may reach on the network, and how it is sandboxed.', at: 23},
  {n: 4, lines: [14, 15, 16, 17], color: C.mesh, title: 'Briefing — the job description', d: 'A short system prompt describing the clusters it works on.', at: 30},
];

const total = YAML.reduce((n, l) => n + l.reduce((m, s) => m + s[1].length, 0) + 1, 0);

export const S04Anatomy: React.FC = () => {
  const frame = useCurrentFrame();
  let budget = Math.floor(interpolate(frame, [sec(1), sec(7.5)], [0, total], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'}));
  const active = PARTS.filter((p) => frame >= sec(p.at)).length;
  const cursorOn = Math.floor(frame / 15) % 2 === 0;

  const rows = YAML.map((line, idx) => {
    const spans = line.map(([cls, txt], j) => {
      const shown = txt.slice(0, Math.max(0, budget));
      budget -= txt.length;
      return (
        <span key={j} style={{color: CLS[cls]}}>
          {shown}
        </span>
      );
    });
    budget -= 1;
    const part = PARTS.find((p) => p.lines.includes(idx));
    const lit = part && part.n <= active;
    return (
      <div
        key={idx}
        style={{
          margin: '0 -34px',
          padding: '0 34px',
          whiteSpace: 'pre',
          minHeight: 34,
          background: lit ? `${part!.color}22` : 'transparent',
          boxShadow: lit ? `inset 6px 0 0 ${part!.color}` : 'none',
        }}
      >
        {spans}
        {budget < 0 && budget > -2 && cursorOn ? <span style={{background: '#79c0ff', color: '#79c0ff'}}>▌</span> : null}
      </div>
    );
  });

  return (
    <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
      <Heading kicker="Chapter 3 · Anatomy of an agent" title="An agent is a Kubernetes object with four parts" />

      <div style={{position: 'absolute', left: 120, top: 230}}>
        <Card style={{width: 860, height: 660, padding: '24px 34px', fontFamily: MONO, fontSize: 23, lineHeight: '34px', overflow: 'hidden'}}>
          {rows}
        </Card>
      </div>

      {PARTS.map((p, i) => {
        const local = frame - sec(p.at);
        const e = interpolate(local, [0, 14], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});
        const isCurrent = p.n === active;
        return (
          <div key={p.n} style={{position: 'absolute', left: 1040, top: 230 + i * 168, opacity: e, transform: `translateX(${(1 - e) * 60}px)`}}>
            <Card accent={p.color} glow={isCurrent ? 0.8 : 0} style={{width: 760, height: 150, padding: '22px 26px', display: 'flex', gap: 22}}>
              <div style={{flex: 'none', width: 58, height: 58, borderRadius: 16, background: p.color, color: C.bg0, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 900, fontSize: 30}}>
                {p.n}
              </div>
              <div>
                <div style={{fontSize: 30, fontWeight: 800}}>{p.title}</div>
                <div style={{fontSize: 22, color: C.dim, marginTop: 6, lineHeight: 1.4}}>{p.d}</div>
              </div>
            </Card>
          </div>
        );
      })}

      <Captions
        lines={[
          {at: 0.5, text: <>Each agent is written down as a short YAML file: a Kubernetes resource of kind <B c={C.cluster2}>Agent</B>.</>},
          {at: 9, text: <>Part one, the <B c={C.cluster2}>model</B>: which brain answers. Here, the local qwen2.5:7b.</>},
          {at: 15, text: <>Part two, the <B c={C.target}>skills</B>. web-endpoint gives the agent an address you can chat with…</>},
          {at: 19, text: <>…and k8s-ops gives it a tool: a sidecar container that can run real <B>kubectl</B> commands.</>},
          {at: 23, text: <>Part three, the <B c={C.policy}>policy</B>: a separate object saying what the agent is allowed to reach.</>},
          {at: 30, text: <>Part four, the <B c={C.mesh}>briefing</B>: a few sentences that tell it which clusters it works on.</>},
          {at: 35.5, text: 'Apply the file with kubectl, and Sympozium turns it into a working agent.'},
        ]}
      />
    </AbsoluteFill>
  );
};
