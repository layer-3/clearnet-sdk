import { useMemo } from "react";

/**
 * A 5x5 mirrored identicon in the brand palette, derived from the address
 * bytes, so the same account always looks the same here.
 */
export function Identicon({ address, size = 32, className }: { address: string; size?: number; className?: string }) {
  const { cells, hue } = useMemo(() => {
    const hex = address.toLowerCase().replace(/^0x/, "").padEnd(40, "0");
    const bits: boolean[] = [];
    for (let i = 0; i < 15; i++) bits.push(parseInt(hex[i + 4]!, 16) % 2 === 0);
    const cells: Array<[number, number]> = [];
    for (let y = 0; y < 5; y++) {
      for (let x = 0; x < 3; x++) {
        if (bits[y * 3 + x]) {
          cells.push([x, y]);
          if (x < 2) cells.push([4 - x, y]);
        }
      }
    }
    return { cells, hue: parseInt(hex.slice(0, 2), 16) % 3 };
  }, [address]);
  const [bg, fg] = [
    ["#fcd000", "#000000"],
    ["#000000", "#fcd000"],
    ["#ffffff", "#000000"],
  ][hue]!;
  return (
    <svg width={size} height={size} viewBox="-1 -1 7 7" className={className} style={{ borderRadius: size / 2, background: bg, flexShrink: 0 }} aria-hidden>
      {cells.map(([x, y]) => (
        <rect key={`${x}-${y}`} x={x} y={y} width={1.02} height={1.02} fill={fg} />
      ))}
    </svg>
  );
}
