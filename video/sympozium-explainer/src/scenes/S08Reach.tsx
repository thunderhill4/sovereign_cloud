import React from 'react';
import {AbsoluteFill, Easing, interpolate, useCurrentFrame} from 'remotion';
import {C, FONT, MONO, sec} from '../theme';
import {B, Captions, Card, Enter, Heading} from '../components/ui';

const ROWS = [
  {agent: 'cluster2-agent', how: 'its own in-cluster identity', target: 'cluster2', what: 'Cluster API · KubeVirt VMs · disk images · nodes', color: C.cluster2, at: 1},
  {agent: 'target-cluster-agent', how: 'a mounted kubeconfig Secret', target: 'target-cluster', what: 'pods · namespaces · nodes — reached via .223, which wakes it up', color: C.target, at: 6},
  {agent: 'mesh-sre-agent', how: 'identity + Istio read access', target: 'Istio ambient mesh', what: 'istiod · ztunnel · Gateways · AuthorizationPolicies', color: C.mesh, at: 11},
];
const Y = [250, 450, 650];

export const S08Reach: React.FC = () => {
  const frame = useCurrentFrame();
  return (
    <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
      <Heading kicker="Chapter 7 · Who can see what" title="Each agent's tools reach exactly one place" />

      <svg width={1920} height={1080} style={{position: 'absolute'}}>
        {ROWS.map((r, i) => {
          const y = Y[i] + 75;
          const draw = interpolate(frame, [sec(r.at + 1), sec(r.at + 2.4)], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp', easing: Easing.inOut(Easing.cubic)});
          const len = 560;
          const started = frame > sec(r.at + 2.4);
          const loop = (((frame - sec(r.at + 2.4)) % 60) + 60) % 60 / 60;
          return (
            <g key={r.agent}>
              <path d={`M640 ${y} L1200 ${y}`} stroke={r.color} strokeWidth={5} fill="none" strokeDasharray={len} strokeDashoffset={len * (1 - draw)} />
              {started && <circle cx={640 + 560 * loop} cy={y} r={10} fill={r.color} opacity={1 - Math.pow(loop, 5)} />}
            </g>
          );
        })}
      </svg>

      {ROWS.map((r, i) => (
        <React.Fragment key={r.agent}>
          <Enter delay={r.at} dx={-40} dy={0} style={{left: 120, top: Y[i]}}>
            <Card accent={r.color} glow={0.4} style={{width: 520, height: 150, padding: '24px 30px'}}>
              <div style={{fontFamily: MONO, fontSize: 30, fontWeight: 800, color: r.color}}>{r.agent}</div>
              <div style={{fontSize: 22, color: C.dim, marginTop: 10}}>via {r.how}</div>
            </Card>
          </Enter>
          <Enter delay={r.at + 2.2} dx={40} dy={0} style={{left: 1200, top: Y[i]}}>
            <Card accent={r.color} style={{width: 600, height: 150, padding: '24px 30px'}}>
              <div style={{fontSize: 30, fontWeight: 800}}>{r.target}</div>
              <div style={{fontSize: 21, color: C.dim, marginTop: 10, lineHeight: 1.4}}>{r.what}</div>
            </Card>
          </Enter>
        </React.Fragment>
      ))}

      <Captions
        lines={[
          {at: 1, text: <><B c={C.cluster2}>cluster2-agent</B> uses its own identity inside cluster2, so it sees the management cluster.</>},
          {at: 6, text: <><B c={C.target}>target-cluster-agent</B> carries a key to the target cluster. Asking it a question can wake that cluster up.</>},
          {at: 11, text: <><B c={C.mesh}>mesh-sre-agent</B> has extra read access to Istio objects, and nothing more.</>},
          {at: 17, text: 'No agent can reach cluster1, the internet, or another agent\'s cluster. Small reach, small risk.'},
        ]}
      />
    </AbsoluteFill>
  );
};
