"use client";

import { clsx } from "clsx";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { useEffect, useState } from "react";
import { useHydrated } from "@/lib/hydration";

export type ClearingStage = "idle" | "signing" | "cosigning" | "sealed" | "rejected" | "unknown";

const NODES = 5;
const QUORUM = 4;
const C = 200;
const R = 128;
const nodes = Array.from({ length: NODES }, (_, i) => {
  const a = ((-90 + i * (360 / NODES)) * Math.PI) / 180;
  return { x: C + R * Math.cos(a), y: C + R * Math.sin(a) };
});
const SENDER = { x: 22, y: 330 };
const RECIPIENT = { x: 378, y: 70 };

/**
 * A shard of clearnodes around one account block. Validators near the account
 * co-sign; at quorum the block is sealed. Drawn in the canvas ink so it works
 * on the yellow page (light) and the black page (dark).
 */
export function Clearing({ stage, legs, className }: { stage: ClearingStage; legs: string[]; className?: string }) {
  const reducedMotion = useReducedMotion();
  const reduce = useHydrated() && reducedMotion;
  const [tick, setTick] = useState<{ stage: ClearingStage; n: number }>({ stage, n: 0 });

  useEffect(() => {
    if (stage !== "cosigning" || reduce) return;
    const id = window.setInterval(() => setTick((t) => ({ stage, n: (t.stage === stage ? t.n : 0) + 1 })), 260);
    return () => window.clearInterval(id);
  }, [stage, reduce]);

  const ticks = tick.stage === stage ? tick.n : 0;
  const lit = stage === "sealed" ? QUORUM : stage === "cosigning" ? (reduce ? QUORUM - 1 : Math.min(ticks, QUORUM - 1)) : 0;

  const packet = stage === "signing" || stage === "idle" ? SENDER : stage === "sealed" ? RECIPIENT : { x: C, y: C };
  const sealed = stage === "sealed";
  const failed = stage === "rejected";
  const ink = "var(--on-canvas)";
  const hero = "var(--accent)";
  const edge = "var(--accent-edge)";
  const onHero = "var(--on-accent)";

  return (
    <svg viewBox="0 0 400 400" className={clsx("block h-auto w-full select-none", className)} role="img" aria-label={label(stage, lit)}>
      {/* Mesh: pentagon and spokes */}
      <g stroke={ink} strokeOpacity={0.16} strokeWidth={1.25} fill="none">
        <circle cx={C} cy={C} r={R + 34} strokeDasharray="2 7" strokeOpacity={0.22} />
        <polygon points={nodes.map((p) => `${p.x},${p.y}`).join(" ")} />
        {nodes.map((p, i) => (
          <line key={i} x1={C} y1={C} x2={p.x} y2={p.y} />
        ))}
      </g>

      {/* Lit spokes */}
      {nodes.map((p, i) => (
        <motion.line
          key={`s${i}`}
          x1={C}
          y1={C}
          x2={p.x}
          y2={p.y}
          stroke={edge}
          strokeWidth={3}
          strokeLinecap="round"
          initial={false}
          animate={{ pathLength: i < lit ? 1 : 0, opacity: i < lit ? 1 : 0 }}
          transition={{ duration: reduce ? 0 : 0.24, ease: [0.2, 0.9, 0.1, 1] }}
        />
      ))}

      {/* Sender -> block -> recipient path */}
      <path d={`M${SENDER.x},${SENDER.y} L${C},${C} L${RECIPIENT.x},${RECIPIENT.y}`} stroke={ink} strokeOpacity={0.28} strokeWidth={1.25} strokeDasharray="1 6" strokeLinecap="round" fill="none" />
      <circle cx={SENDER.x} cy={SENDER.y} r={5} fill={ink} />
      <circle cx={RECIPIENT.x} cy={RECIPIENT.y} r={5} fill="none" stroke={ink} strokeWidth={2} />
      <g fill={ink} fontSize="13" fontWeight={600} style={{ fontVariationSettings: '"wdth" 88' }}>
        <text x={SENDER.x + 12} y={SENDER.y + 22}>You</text>
        <text x={RECIPIENT.x - 12} y={RECIPIENT.y - 14} textAnchor="end">Recipient</text>
      </g>

      {/* Clearnodes */}
      {nodes.map((p, i) => {
        const on = i < lit;
        return (
          <g key={`n${i}`} transform={`translate(${p.x} ${p.y})`}>
            <motion.circle r={17} initial={false} animate={{ fill: on ? hero : "var(--canvas)", scale: on && !reduce ? [1, 1.18, 1] : 1 }} transition={{ duration: 0.3 }} stroke={on ? edge : ink} strokeWidth={2} />
            <AnimatePresence>
              {on && (
                <motion.path
                  d="M-6 0.5 -2 4.5 6.5 -4"
                  stroke={onHero}
                  strokeWidth={2.6}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  fill="none"
                  initial={{ pathLength: 0 }}
                  animate={{ pathLength: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: reduce ? 0 : 0.22 }}
                />
              )}
            </AnimatePresence>
          </g>
        );
      })}

      {/* The signed operation */}
      <motion.g initial={false} animate={{ x: packet.x, y: packet.y, opacity: stage === "rejected" ? 0 : 1 }} transition={{ duration: reduce ? 0 : stage === "sealed" ? 0.7 : 0.55, ease: [0.2, 0.9, 0.1, 1] }}>
        <circle r={9} fill={hero} stroke={edge} strokeWidth={2.5} />
        {stage === "signing" && !reduce && (
          <motion.circle r={9} fill="none" stroke={edge} strokeWidth={2} animate={{ r: [9, 22], opacity: [0.8, 0] }} transition={{ duration: 1.2, repeat: Infinity }} />
        )}
      </motion.g>
      {/* Account block */}
      <motion.g initial={false} animate={{ scale: sealed && !reduce ? [1, 1.06, 1] : 1 }} transition={{ duration: 0.4 }} style={{ transformOrigin: `${C}px ${C}px` }}>
        <rect x={C - 58} y={C - 44} width={116} height={88} rx={14} fill={failed ? "var(--flare)" : sealed ? hero : "var(--canvas)"} stroke={failed ? "var(--flare)" : sealed ? edge : ink} strokeWidth={2.5} />
        <foreignObject x={C - 58} y={C - 44} width={116} height={88}>
          <div className="flex h-full w-full flex-col items-center justify-center px-2 text-center" style={{ color: sealed || failed ? onHero : ink }}>
            <AnimatePresence mode="wait" initial={false}>
              <motion.div key={sealed ? "sealed" : failed ? "failed" : legs.join("+")} initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -6 }} transition={{ duration: reduce ? 0 : 0.18 }}>
                {sealed ? (
                  <span className="type-title block text-[24px]">Sealed</span>
                ) : failed ? (
                  <span className="type-title block text-[22px] text-white">Rejected</span>
                ) : (
                  legs.slice(0, 2).map((l) => (
                    <span key={l} className="type-title block truncate text-[15px] leading-tight">
                      {l}
                    </span>
                  ))
                )}
              </motion.div>
            </AnimatePresence>
          </div>
        </foreignObject>
      </motion.g>

    </svg>
  );
}

function label(stage: ClearingStage, lit: number) {
  switch (stage) {
    case "signing":
      return "Waiting for your signature";
    case "cosigning":
      return `Validators co-signing: ${lit} of ${QUORUM} needed`;
    case "sealed":
      return `Sealed by ${QUORUM} of ${NODES} validators`;
    case "rejected":
      return "Rejected by the network";
    default:
      return "A shard of five validators around your account";
  }
}

const SCENES = [["250 USDC"], ["0.8 ETH", "+ 40 YELLOW"], ["1,200 YELLOW"]];
const STEPS: Array<[ClearingStage, number]> = [
  ["idle", 900],
  ["signing", 1100],
  ["cosigning", 1500],
  ["sealed", 2200],
];

/** Self-running showcase for the connect screen. Click to replay with another transfer. */
export function ClearingShowcase({ className }: { className?: string }) {
  const reducedMotion = useReducedMotion();
  const reduce = useHydrated() && reducedMotion;
  const [scene, setScene] = useState(0);
  const [step, setStep] = useState(0);

  useEffect(() => {
    if (reduce) return;
    const id = window.setTimeout(() => {
      if (step === STEPS.length - 1) {
        setScene((s) => (s + 1) % SCENES.length);
        setStep(0);
      } else setStep(step + 1);
    }, STEPS[step]![1]);
    return () => window.clearTimeout(id);
  }, [step, reduce]);

  const stage = reduce ? "sealed" : STEPS[step]![0];
  const caption =
    stage === "signing" ? "You sign once" : stage === "cosigning" ? "Validators near your account co-sign" : stage === "sealed" ? "Sealed. The transfer is final." : "A transfer, ready to clear";

  return (
    <figure className={className}>
      <button
        type="button"
        className="block w-full rounded-sheet focus-visible:outline-offset-4"
        onClick={() => {
          setScene((s) => (s + 1) % SCENES.length);
          setStep(1);
        }}
        aria-label="Replay the clearing illustration"
      >
        <Clearing stage={stage} legs={SCENES[scene]!} />
      </button>
      <figcaption className="mt-2 flex items-center justify-center gap-2 text-[15px] text-on-canvas-muted" aria-live="polite">
        <span className={clsx("size-2 rounded-full bg-on-canvas dark:bg-accent", stage !== "sealed" && "animate-[pulse-dot_1.2s_ease-in-out_infinite]")} />
        {caption}
      </figcaption>
    </figure>
  );
}
