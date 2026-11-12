import React from 'react';
import {AbsoluteFill, useCurrentFrame} from 'remotion';
import {C, FONT, MONO} from '../theme';
import {B, Captions, Card, Chip, Enter, Heading, useEnter} from '../components/ui';

const ClusterBox: React.FC<{
  delay: number;
  x: number;
  w: number;
  color: string;
  title: string;
  kind: string;
  items: string[];
  glow?: number;
  children?: React.ReactNode;
}> = ({delay, x, w, color, title, kind, items, glow = 0, children}) => (
  <Enter delay={delay} scale style={{left: x, top: 300}}>
    <Card accent={color} glow={glow} style={{width: w, height: 360, padding: '26px 30px', position: 'relative'}}>
      <div style={{fontSize: 18, letterSpacing: 3, color, fontWeight: 800, textTransform: 'uppercase'}}>{kind}</div>
      <div style={{fontSize: 38, fontWeight: 800, marginTop: 6, fontFamily: MONO, letterSpacing: -1}}>{title}</div>
      <div style={{marginTop: 10}}>
        {items.map((i) => (
          <Chip key={i} color={color}>
            {i}
          </Chip>
        ))}
      </div>
      {children}
    </Card>
  </Enter>
);

export const S02Platform: React.FC = () => {
  const frame = useCurrentFrame();
  const host = useEnter(0.6);
  const sym = useEnter(15, 11);
  const glow = frame > 450 ? 0.6 + 0.4 * Math.sin(frame / 10) : 0;
  const olla = useEnter(21);
  return (
    <AbsoluteFill style={{fontFamily: FONT, color: C.text}}>
      <Heading kicker="Chapter 1 · The platform" title="Where do the agents live?" />

      {/* the host machine */}
      <div
        style={{
          position: 'absolute',
          left: 100,
          top: 250,
          width: 1720,
          height: 610,
          borderRadius: 28,
          border: `2px dashed ${C.border}`,
          opacity: host,
        }}
      >
        <div style={{position: 'absolute', left: 26, top: -16, background: C.bg1, padding: '0 14px', fontSize: 20, color: C.faint, letterSpacing: 3, fontWeight: 700}}>
          ONE LINUX MACHINE
        </div>
      </div>

      <ClusterBox
        delay={2}
        x={150}
        w={460}
        color={C.cluster1}
        kind="Kind cluster"
        title="cluster1"
        items={['Istio demo mesh', 'Gateway API']}
      />
      <ClusterBox
        delay={5}
        x={650}
        w={680}
        color={C.cluster2}
        kind="Kind cluster · management"
        title="cluster2"
        items={['Cluster API', 'KubeVirt VMs', 'MetalLB', 'CDI images']}
        glow={glow}
      >
        <div
          style={{
            position: 'absolute',
            left: 30,
            bottom: 26,
            right: 30,
            height: 92,
            borderRadius: 16,
            background: 'linear-gradient(90deg,#1f3b73,#3b2a6b)',
            border: '2px solid #a371f7',
            display: 'flex',
            alignItems: 'center',
            gap: 18,
            padding: '0 24px',
            opacity: sym,
            transform: `translateY(${(1 - sym) * -160}px) scale(${0.8 + 0.2 * sym})`,
          }}
        >
          <span style={{fontSize: 44}}>🤖</span>
          <div>
            <div style={{fontSize: 30, fontWeight: 800}}>Sympozium</div>
            <div style={{fontSize: 19, color: C.dim}}>runs the AI agents · namespace <span style={{fontFamily: MONO}}>sympozium-system</span></div>
          </div>
        </div>
      </ClusterBox>
      <ClusterBox
        delay={9}
        x={1370}
        w={410}
        color={C.target}
        kind="k3s · nodes are VMs"
        title="target-cluster"
        items={['your workloads', 'built on demand']}
      />

      {/* VM link */}
      <Enter delay={10} dy={0} dx={-20} style={{left: 1300, top: 410}}>
        <div style={{fontSize: 38, color: C.target}}>⇢</div>
      </Enter>

      <div
        style={{
          position: 'absolute',
          left: 150,
          top: 690,
          width: 1630,
          height: 140,
          opacity: olla,
          transform: `translateY(${(1 - olla) * 30}px)`,
        }}
      >
        <Card accent={C.model} glow={olla} style={{width: '100%', height: '100%', display: 'flex', alignItems: 'center', gap: 28, padding: '0 36px'}}>
          <span style={{fontSize: 60}}>🧠</span>
          <div>
            <div style={{fontSize: 32, fontWeight: 800}}>
              Ollama on the host — the language model, <B>qwen2.5:7b</B>
            </div>
            <div style={{fontSize: 22, color: C.dim, marginTop: 6}}>
              Runs on this machine itself, at <span style={{fontFamily: MONO}}>172.18.0.1:11434</span>. Questions and cluster data never leave the box.
            </div>
          </div>
        </Card>
      </div>

      <Captions
        lines={[
          {at: 1, text: 'Everything runs on one Linux machine, as Kubernetes clusters inside containers.'},
          {at: 5, text: <>cluster2 is the <B c={C.cluster2}>management cluster</B>. It builds other clusters out of virtual machines.</>},
          {at: 9.5, text: <>The <B c={C.target}>target-cluster</B> is one of those: a small k3s cluster whose nodes are VMs on cluster2.</>},
          {at: 15, text: <><B c={C.mesh}>Sympozium</B> is installed in cluster2. It is the system that runs and manages the AI agents.</>},
          {at: 21, text: <>The "brain" is a local model, <B>qwen2.5:7b</B>, served by Ollama. No cloud AI service is involved.</>},
        ]}
      />
    </AbsoluteFill>
  );
};
