import React from 'react';
import {AbsoluteFill} from 'remotion';
import {C, FONT, MONO} from '../theme';
import {Card, Enter} from '../components/ui';

const POINTS = [
  {c: C.cluster2, t: <>An agent = <b>model</b> + <b>skills</b> + <b>policy</b> + <b>briefing</b>, in one YAML file.</>},
  {c: C.target, t: <>Every chat runs in its own short-lived pod, using real <span style={{fontFamily: MONO, color: C.model}}>kubectl</span>.</>},
  {c: C.policy, t: <>Guard, network policy and RBAC set the limits — not the prompt.</>},
];

export const S10Recap: React.FC = () => (
  <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
    <Enter delay={0.2} dy={20} style={{left: 260, top: 150}}>
      <div style={{fontSize: 22, letterSpacing: 6, textTransform: 'uppercase', color: C.faint, fontWeight: 700}}>Recap</div>
    </Enter>
    {POINTS.map((p, i) => (
      <Enter key={i} delay={0.6 + i * 1.2} dy={40} style={{left: 260, top: 210 + i * 150}}>
        <Card accent={p.c} glow={0.4} style={{width: 1400, height: 120, display: 'flex', alignItems: 'center', gap: 28, padding: '0 34px'}}>
          <div style={{width: 64, height: 64, borderRadius: 18, background: p.c, color: C.bg0, fontWeight: 900, fontSize: 34, display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
            {i + 1}
          </div>
          <div style={{fontSize: 34, fontWeight: 700}}>{p.t}</div>
        </Card>
      </Enter>
    ))}
    <Enter delay={4.5} dy={20} style={{left: 260, top: 700}}>
      <div style={{fontFamily: MONO, fontSize: 28, color: C.dim, lineHeight: 1.8}}>
        <span style={{color: C.faint}}>start here →</span> 06-sympozium/ · make sympozium-install · sovctl ai agents
      </div>
    </Enter>
  </AbsoluteFill>
);
