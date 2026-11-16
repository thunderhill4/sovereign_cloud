import React from 'react';
import {AbsoluteFill, useCurrentFrame} from 'remotion';
import {C, FONT, MONO} from '../theme';
import {Captions, Enter, useEnter} from '../components/ui';

const ORBIT = [
  {name: 'cluster2-agent', color: C.cluster2},
  {name: 'target-cluster-agent', color: C.target},
  {name: 'mesh-sre-agent', color: C.mesh},
];

const Hub: React.FC = () => {
  const frame = useCurrentFrame();
  const p = useEnter(1.2, 14);
  const cx = 1460;
  const cy = 470;
  return (
    <div style={{position: 'absolute', left: 0, top: 0, opacity: p}}>
      <svg width={1920} height={1080} style={{position: 'absolute'}}>
        <circle cx={cx} cy={cy} r={230} fill="none" stroke="#2a3755" strokeWidth={2} strokeDasharray="6 10" />
        <circle cx={cx} cy={cy} r={110 + 8 * Math.sin(frame / 18)} fill="#58a6ff10" stroke="#58a6ff55" strokeWidth={2} />
      </svg>
      <div
        style={{
          position: 'absolute',
          left: cx - 90,
          top: cy - 90,
          width: 180,
          height: 180,
          borderRadius: 90,
          background: 'linear-gradient(135deg,#1f3b73,#3b2a6b)',
          border: '2px solid #58a6ff88',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 76,
        }}
      >
        🧠
      </div>
      {ORBIT.map((o, i) => {
        const a = frame / 80 + (i * Math.PI * 2) / 3;
        const x = cx + Math.cos(a) * 230;
        const y = cy + Math.sin(a) * 230;
        return (
          <div
            key={o.name}
            style={{
              position: 'absolute',
              left: x - 14,
              top: y - 14,
              width: 28,
              height: 28,
              borderRadius: 14,
              background: o.color,
              boxShadow: `0 0 30px ${o.color}`,
            }}
          >
            <div
              style={{
                position: 'absolute',
                left: 36,
                top: -2,
                whiteSpace: 'nowrap',
                fontFamily: MONO,
                fontSize: 22,
                color: o.color,
              }}
            >
              {o.name}
            </div>
          </div>
        );
      })}
    </div>
  );
};

export const S01Title: React.FC = () => (
  <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
    <Hub />
    <Enter delay={0.3} dy={20} style={{left: 140, top: 270}}>
      <div style={{fontSize: 22, letterSpacing: 6, textTransform: 'uppercase', color: C.faint, fontWeight: 700}}>
        Sovereign Cloud × Sympozium
      </div>
    </Enter>
    <Enter delay={0.7} dy={50} style={{left: 140, top: 320, width: 1050}}>
      <div style={{fontSize: 92, fontWeight: 900, letterSpacing: -2.5, lineHeight: 1.04}}>
        How{' '}
        <span
          style={{
            background: 'linear-gradient(90deg,#58a6ff,#a371f7 55%,#f778ba)',
            WebkitBackgroundClip: 'text',
            color: 'transparent',
          }}
        >
          AI agents
        </span>
        <br />
        run your clusters
      </div>
    </Enter>
    <Enter delay={1.8} dy={30} style={{left: 140, top: 560, width: 950}}>
      <div style={{fontSize: 32, color: C.dim, lineHeight: 1.45}}>
        A plain-English tour of the Sympozium agents in this platform: what they are, how a question
        reaches them, and what keeps them safe.
      </div>
    </Enter>
    <Captions
      lines={[
        {at: 3, text: 'This platform has AI helpers built in. They are called Sympozium agents.'},
        {at: 9, text: 'You ask them questions in plain English, and they look at your clusters to answer.'},
        {at: 14.5, text: 'In the next five minutes: what they are, how a question travels, and what keeps them safe.'},
      ]}
    />
  </AbsoluteFill>
);
