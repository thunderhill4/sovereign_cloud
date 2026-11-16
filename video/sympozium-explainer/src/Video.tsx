import React from 'react';
import {AbsoluteFill, interpolate, useCurrentFrame} from 'remotion';
import {TransitionSeries, linearTiming} from '@remotion/transitions';
import {fade} from '@remotion/transitions/fade';
import {slide} from '@remotion/transitions/slide';
import {C, FONT, sec} from './theme';
import {Background} from './components/ui';
import {S01Title} from './scenes/S01Title';
import {S02Platform} from './scenes/S02Platform';
import {S03Agents} from './scenes/S03Agents';
import {S04Anatomy} from './scenes/S04Anatomy';
import {S05Journey} from './scenes/S05Journey';
import {S06Runs} from './scenes/S06Runs';
import {S07Guardrails} from './scenes/S07Guardrails';
import {S08Reach} from './scenes/S08Reach';
import {S09Verify} from './scenes/S09Verify';
import {S10Recap} from './scenes/S10Recap';

// Seconds each chapter is on screen. Sums to 300 (5:00).
export const CHAPTERS: {title: string; secs: number; C: React.FC}[] = [
  {title: 'Intro', secs: 20, C: S01Title},
  {title: 'The platform', secs: 30, C: S02Platform},
  {title: 'Meet the agents', secs: 30, C: S03Agents},
  {title: 'Anatomy', secs: 40, C: S04Anatomy},
  {title: 'One question', secs: 60, C: S05Journey},
  {title: 'Two kinds of pods', secs: 30, C: S06Runs},
  {title: 'Guardrails', secs: 35, C: S07Guardrails},
  {title: 'Who sees what', secs: 25, C: S08Reach},
  {title: 'Trust, but verify', secs: 20, C: S09Verify},
  {title: 'Recap', secs: 10, C: S10Recap},
];

// Transitions overlap neighbouring scenes; each non-final scene is padded by
// this much so the visible chapter lengths still add up to exactly 300s.
const T = 20;
export const TOTAL_FRAMES = CHAPTERS.reduce((n, c) => n + sec(c.secs), 0);

const starts = CHAPTERS.reduce<number[]>((acc, c, i) => [...acc, i === 0 ? 0 : acc[i - 1] + sec(CHAPTERS[i - 1].secs)], []);

const Progress: React.FC = () => {
  const frame = useCurrentFrame();
  let idx = 0;
  starts.forEach((s, i) => {
    if (frame >= s) idx = i;
  });
  const fadeIn = interpolate(frame, [0, 30], [0, 1], {extrapolateRight: 'clamp'});
  return (
    <AbsoluteFill style={{fontFamily: FONT, opacity: fadeIn}}>
      <div style={{position: 'absolute', left: 120, right: 120, bottom: 30, display: 'flex', gap: 8}}>
        {CHAPTERS.map((c, i) => {
          const p = interpolate(frame, [starts[i], starts[i] + sec(c.secs)], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});
          return (
            <div key={c.title} style={{flex: c.secs, height: 6, borderRadius: 3, background: '#1a2440', overflow: 'hidden'}}>
              <div style={{width: `${p * 100}%`, height: '100%', background: i === idx ? 'linear-gradient(90deg,#58a6ff,#a371f7)' : '#3a4a70'}} />
            </div>
          );
        })}
      </div>
      <div style={{position: 'absolute', right: 120, top: 44, fontSize: 18, letterSpacing: 3, color: C.faint, fontWeight: 700, textTransform: 'uppercase'}}>
        {String(idx + 1).padStart(2, '0')} / {CHAPTERS.length} · {CHAPTERS[idx].title}
      </div>
    </AbsoluteFill>
  );
};

export const Video: React.FC = () => (
  <AbsoluteFill style={{background: C.bg0}}>
    <Background />
    <TransitionSeries>
      {CHAPTERS.flatMap((c, i) => {
        const last = i === CHAPTERS.length - 1;
        const items = [
          <TransitionSeries.Sequence key={c.title} durationInFrames={sec(c.secs) + (last ? 0 : T)}>
            <c.C />
          </TransitionSeries.Sequence>,
        ];
        if (!last) {
          items.push(
            <TransitionSeries.Transition
              key={c.title + '-t'}
              presentation={i % 2 ? slide({direction: 'from-right'}) : fade()}
              timing={linearTiming({durationInFrames: T})}
            />,
          );
        }
        return items;
      })}
    </TransitionSeries>
    <Progress />
  </AbsoluteFill>
);
