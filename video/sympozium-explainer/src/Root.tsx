import React from 'react';
import {Composition} from 'remotion';
import {FPS, HEIGHT, WIDTH} from './theme';
import {TOTAL_FRAMES, Video} from './Video';

export const Root: React.FC = () => (
  <Composition id="SympoziumExplainer" component={Video} durationInFrames={TOTAL_FRAMES} fps={FPS} width={WIDTH} height={HEIGHT} />
);
