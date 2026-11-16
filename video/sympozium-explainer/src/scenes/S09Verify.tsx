import React from 'react';
import {AbsoluteFill} from 'remotion';
import {C, FONT, MONO} from '../theme';
import {B, Captions, Card, Enter, Heading} from '../components/ui';

const RUNS = [
  {n: 1, ok: true, calls: 1, what: 'ran kubectl, answered from real output'},
  {n: 2, ok: false, calls: 0, what: 'printed the kubectl command as text'},
  {n: 3, ok: false, calls: 0, what: 'made up plausible-looking output'},
];

export const S09Verify: React.FC = () => (
  <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
    <Heading kicker="Chapter 8 · Trust, but verify" title="Small models sometimes fake it" />

    {RUNS.map((r, i) => {
      const color = r.ok ? C.target : C.danger;
      return (
        <Enter key={r.n} delay={1 + i * 2.2} dx={-40} dy={0} style={{left: 120, top: 250 + i * 170}}>
          <Card accent={color} glow={0.4} style={{width: 860, height: 140, padding: '0 32px', display: 'flex', alignItems: 'center', gap: 26}}>
            <div style={{fontSize: 52, color}}>{r.ok ? '✓' : '✗'}</div>
            <div>
              <div style={{fontSize: 27, fontWeight: 800}}>Same question, run {r.n}</div>
              <div style={{fontSize: 22, color: C.dim, marginTop: 6}}>{r.what}</div>
            </div>
            <div style={{marginLeft: 'auto', fontFamily: MONO, fontSize: 24, color}}>toolCalls: {r.calls}</div>
          </Card>
        </Enter>
      );
    })}

    <Enter delay={8.5} scale style={{left: 1060, top: 250}}>
      <Card accent={C.model} glow={0.6} style={{width: 740, height: 480, padding: '32px 36px'}}>
        <div style={{fontSize: 20, letterSpacing: 3, color: C.model, fontWeight: 800}}>HOW TO CHECK</div>
        <div style={{fontSize: 30, fontWeight: 800, marginTop: 12, lineHeight: 1.3}}>Look at what the run did, not what it said</div>
        <div style={{fontFamily: MONO, fontSize: 21, color: C.text, background: '#0b1222', border: `1px solid ${C.border}`, borderRadius: 14, padding: '18px 20px', marginTop: 24, lineHeight: 1.55}}>
          <span style={{color: C.target}}>$</span> kubectl get agentrun &lt;run&gt; \<br />
          &nbsp;&nbsp;-n sympozium-system -o yaml
          <br />
          <span style={{color: C.faint}}># status.tokenUsage.toolCalls</span>
        </div>
        <div style={{fontSize: 23, color: C.dim, marginTop: 24, lineHeight: 1.45}}>
          <b style={{color: C.text}}>0</b> means no tool ran: the answer is a guess.
        </div>
      </Card>
    </Enter>

    <Captions
      lines={[
        {at: 1, text: <>A 7-billion-parameter model is small. Asked the same question three times, it used its tool <B>once</B>.</>},
        {at: 8.5, text: <>So never judge by the prose. Each run records how many tools it really called: <B>toolCalls</B>.</>},
        {at: 14.5, text: 'If that number is zero, the agent did not look. Treat the answer as a guess.'},
      ]}
    />
  </AbsoluteFill>
);
