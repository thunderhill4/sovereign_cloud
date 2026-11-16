import React from 'react';
import {AbsoluteFill, useCurrentFrame} from 'remotion';
import {C, FONT, MONO, sec} from '../theme';
import {B, Captions, Card, Chip, Enter, Heading, useRange} from '../components/ui';

const AGENTS = [
  {
    name: 'cluster2-agent',
    ip: '172.18.255.213',
    icon: '🛠️',
    color: C.cluster2,
    job: 'Looks after the management cluster: Cluster API, KubeVirt VMs, disk images, nodes.',
    ask: '“Which VMs are running?”',
    delay: 1,
  },
  {
    name: 'target-cluster-agent',
    ip: '172.18.255.214',
    icon: '☸️',
    color: C.target,
    job: 'Works inside the target cluster: its pods, namespaces and nodes.',
    ask: '“What is crashing in my app?”',
    delay: 6,
  },
  {
    name: 'mesh-sre-agent',
    ip: '172.18.255.222',
    icon: '🕸️',
    color: C.mesh,
    job: 'Checks the Istio service mesh: istiod, ztunnel, gateways, policies.',
    ask: '“Is the mesh healthy?”',
    delay: 11,
  },
];

const X = [120, 700, 1280];

export const S03Agents: React.FC = () => {
  const frame = useCurrentFrame();
  const link = useRange(17, 19.5);
  return (
    <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
      <Heading kicker="Chapter 2 · Meet the agents" title="Three specialists, one shared brain" />

      <svg width={1920} height={1080} style={{position: 'absolute'}}>
        {X.map((x, i) => {
          const x1 = x + 260;
          const y1 = 590;
          const x2 = 960;
          const y2 = 700;
          const len = Math.hypot(x2 - x1, y2 - y1);
          const pulse = ((frame - sec(19.5) - i * 8) % 50) / 50;
          return (
            <g key={i}>
              <line
                x1={x1}
                y1={y1}
                x2={x2}
                y2={y2}
                stroke={AGENTS[i].color}
                strokeWidth={4}
                strokeDasharray={len}
                strokeDashoffset={len * (1 - link)}
              />
              {frame > sec(19.5) && (
                <circle
                  cx={x2 + (x1 - x2) * pulse}
                  cy={y2 + (y1 - y2) * pulse}
                  r={8}
                  fill={AGENTS[i].color}
                  opacity={1 - pulse}
                />
              )}
            </g>
          );
        })}
      </svg>

      {AGENTS.map((a, i) => (
        <Enter key={a.name} delay={a.delay} scale dy={60} style={{left: X[i], top: 240}}>
          <Card accent={a.color} glow={frame > sec(a.delay) && frame < sec(a.delay + 5) ? 0.7 : 0} style={{width: 520, height: 350, padding: '28px 32px', position: 'relative'}}>
            <div style={{display: 'flex', alignItems: 'center', gap: 18}}>
              <div style={{width: 66, height: 66, borderRadius: 18, background: `${a.color}26`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 36}}>
                {a.icon}
              </div>
              <div style={{fontFamily: MONO, fontSize: 18, color: a.color, padding: '6px 12px', borderRadius: 99, background: `${a.color}1f`}}>{a.ip}</div>
            </div>
            <div style={{fontFamily: MONO, fontSize: 31, fontWeight: 800, color: a.color, marginTop: 20, letterSpacing: -1}}>{a.name}</div>
            <div style={{fontSize: 23, color: C.dim, marginTop: 12, lineHeight: 1.42}}>{a.job}</div>
            <div style={{fontSize: 23, color: C.text, marginTop: 14, fontStyle: 'italic'}}>{a.ask}</div>
          </Card>
        </Enter>
      ))}

      <Enter delay={16} dy={30} style={{left: 560, top: 700}}>
        <Card accent={C.model} glow={link} style={{width: 800, height: 140, display: 'flex', alignItems: 'center', gap: 24, padding: '0 34px'}}>
          <span style={{fontSize: 56}}>🧠</span>
          <div>
            <div style={{fontSize: 30, fontWeight: 800}}>
              All three think with <B>qwen2.5:7b</B>
            </div>
            <div style={{marginTop: 2}}>
              <Chip color={C.model}>same model · different briefing · different tools</Chip>
            </div>
          </div>
        </Card>
      </Enter>

      <Captions
        lines={[
          {at: 1, text: <><B c={C.cluster2}>cluster2-agent</B> knows the management cluster: the VMs, the images, the cluster builder.</>},
          {at: 6, text: <><B c={C.target}>target-cluster-agent</B> reaches inside the target cluster, where your applications run.</>},
          {at: 11, text: <><B c={C.mesh}>mesh-sre-agent</B> is the networking expert. It checks the Istio service mesh.</>},
          {at: 16.5, text: 'Each one has its own address on the network, so the dashboard can talk to it directly.'},
          {at: 22, text: 'They share the same model. What makes them different is their briefing and their tools.'},
        ]}
      />
    </AbsoluteFill>
  );
};
