export const FPS = 30;
export const WIDTH = 1920;
export const HEIGHT = 1080;

export const C = {
  bg0: '#070a12',
  bg1: '#0c1426',
  bg2: '#13213f',
  card: 'rgba(20,29,49,0.92)',
  border: '#2a3755',
  text: '#e6edf3',
  dim: '#9fb0cf',
  faint: '#5d6b88',
  // one colour per actor, used consistently across scenes
  cluster2: '#58a6ff',
  target: '#56d364',
  mesh: '#d2a8ff',
  policy: '#f0883e',
  model: '#ffa657',
  danger: '#ff7b72',
  cluster1: '#39c5cf',
};

export const FONT = "'Inter','DejaVu Sans','Liberation Sans',sans-serif";
export const MONO = "'DejaVu Sans Mono','Liberation Mono',monospace";

export const sec = (s: number) => Math.round(s * FPS);
