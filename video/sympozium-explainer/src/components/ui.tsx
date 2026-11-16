import React from 'react';
import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
  Easing,
} from 'remotion';
import {C, FONT, MONO, sec} from '../theme';

/** 0→1 spring starting at `delay` seconds. */
export const useEnter = (delay: number, damping = 18) => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  return spring({frame: frame - sec(delay), fps, config: {damping, mass: 0.8}});
};

/** Linear-eased 0→1 between two times in seconds. */
export const useRange = (from: number, to: number, ease = true) => {
  const frame = useCurrentFrame();
  return interpolate(frame, [sec(from), sec(to)], [0, 1], {
    extrapolateLeft: 'clamp',
    extrapolateRight: 'clamp',
    easing: ease ? Easing.inOut(Easing.cubic) : undefined,
  });
};

export const Background: React.FC = () => {
  const frame = useCurrentFrame();
  const drift = (frame * 0.25) % 60;
  return (
    <AbsoluteFill
      style={{
        background: `radial-gradient(1400px 800px at 18% 8%, ${C.bg2} 0%, ${C.bg1} 50%, ${C.bg0} 100%)`,
      }}
    >
      <AbsoluteFill
        style={{
          opacity: 0.16,
          backgroundImage: `linear-gradient(#22304f 1px, transparent 1px), linear-gradient(90deg, #22304f 1px, transparent 1px)`,
          backgroundSize: '60px 60px',
          backgroundPosition: `${drift}px ${drift}px`,
        }}
      />
    </AbsoluteFill>
  );
};

/** Fades + rises in at `delay` seconds. */
export const Enter: React.FC<{
  delay: number;
  dy?: number;
  dx?: number;
  scale?: boolean;
  style?: React.CSSProperties;
  children: React.ReactNode;
}> = ({delay, dy = 40, dx = 0, scale, style, children}) => {
  const p = useEnter(delay);
  return (
    <div
      style={{
        position: 'absolute',
        opacity: p,
        transform: `translate(${(1 - p) * dx}px, ${(1 - p) * dy}px)` + (scale ? ` scale(${0.9 + 0.1 * p})` : ''),
        ...style,
      }}
    >
      {children}
    </div>
  );
};

export const Card: React.FC<{
  style?: React.CSSProperties;
  accent?: string;
  glow?: number;
  children: React.ReactNode;
}> = ({style, accent, glow = 0, children}) => (
  <div
    style={{
      background: C.card,
      border: `2px solid ${glow > 0 && accent ? accent : C.border}`,
      borderRadius: 22,
      boxShadow:
        `0 24px 70px rgba(0,0,0,0.45)` +
        (glow > 0 && accent ? `, 0 0 ${40 * glow}px ${accent}66` : ''),
      fontFamily: FONT,
      color: C.text,
      ...style,
    }}
  >
    {children}
  </div>
);

export const Code: React.FC<{color?: string; children: React.ReactNode; size?: number}> = ({
  color = C.model,
  size,
  children,
}) => (
  <span
    style={{
      fontFamily: MONO,
      color,
      background: '#1b2742',
      padding: '2px 10px',
      borderRadius: 8,
      fontSize: size ? size : '0.88em',
    }}
  >
    {children}
  </span>
);

export const Chip: React.FC<{color: string; children: React.ReactNode}> = ({color, children}) => (
  <span
    style={{
      display: 'inline-block',
      fontFamily: MONO,
      fontSize: 20,
      padding: '6px 14px',
      borderRadius: 10,
      marginRight: 10,
      marginTop: 12,
      background: `${color}1f`,
      border: `1px solid ${color}88`,
      color,
    }}
  >
    {children}
  </span>
);

export const Heading: React.FC<{kicker: string; title: React.ReactNode}> = ({kicker, title}) => {
  const p = useEnter(0.2);
  return (
    <div
      style={{
        position: 'absolute',
        left: 120,
        top: 96,
        opacity: p,
        transform: `translateY(${(1 - p) * -20}px)`,
        fontFamily: FONT,
      }}
    >
      <div style={{fontSize: 20, letterSpacing: 5, textTransform: 'uppercase', color: C.faint, fontWeight: 700}}>
        {kicker}
      </div>
      <div style={{fontSize: 54, fontWeight: 800, color: C.text, letterSpacing: -1, marginTop: 8}}>{title}</div>
    </div>
  );
};

export type Line = {at: number; text: React.ReactNode};

/** Narration subtitle bar: shows the latest line whose `at` (seconds) has passed. */
export const Captions: React.FC<{lines: Line[]}> = ({lines}) => {
  const frame = useCurrentFrame();
  const t = frame / 30;
  let idx = -1;
  lines.forEach((l, i) => {
    if (t >= l.at) idx = i;
  });
  if (idx < 0) return null;
  const local = frame - sec(lines[idx].at);
  const p = interpolate(local, [0, 10], [0, 1], {extrapolateRight: 'clamp'});
  return (
    <div
      style={{
        position: 'absolute',
        left: 0,
        right: 0,
        bottom: 70,
        display: 'flex',
        justifyContent: 'center',
        fontFamily: FONT,
      }}
    >
      <div
        style={{
          maxWidth: 1500,
          textAlign: 'center',
          fontSize: 34,
          lineHeight: 1.4,
          fontWeight: 600,
          color: C.text,
          background: 'rgba(5,8,15,0.78)',
          border: `1px solid ${C.border}`,
          borderRadius: 18,
          padding: '18px 36px',
          opacity: p,
          transform: `translateY(${(1 - p) * 12}px)`,
        }}
      >
        {lines[idx].text}
      </div>
    </div>
  );
};

export const B: React.FC<{c?: string; children: React.ReactNode}> = ({c = C.model, children}) => (
  <b style={{color: c}}>{children}</b>
);
