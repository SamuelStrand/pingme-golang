import type { TargetStatus, TargetTimelinePoint } from "../types";

const STRIP_BLOCKS = 90;

type StripBlock = "up" | "down" | "unknown";

function blocksFromTimeline(timeline: TargetTimelinePoint[]): StripBlock[] {
  if (timeline.length === 0) {
    return Array(STRIP_BLOCKS).fill("unknown");
  }

  const sorted = [...timeline].sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
  );
  const blocks: StripBlock[] = [];

  for (let index = 0; index < STRIP_BLOCKS; index += 1) {
    const ratio = (index + 0.5) / STRIP_BLOCKS;
    const pointIndex = Math.min(sorted.length - 1, Math.floor(ratio * sorted.length));
    blocks.push(sorted[pointIndex].success ? "up" : "down");
  }

  return blocks;
}

function blocksFromStatus(status: TargetStatus): StripBlock[] {
  if (status === "up") {
    return Array(STRIP_BLOCKS).fill("up");
  }
  if (status === "down") {
    return Array(STRIP_BLOCKS).fill("down");
  }
  return Array(STRIP_BLOCKS).fill("unknown");
}

export function UptimeStrip({
  timeline,
  status,
  title = "Recent uptime"
}: {
  timeline?: TargetTimelinePoint[];
  status: TargetStatus;
  title?: string;
}) {
  const blocks = timeline && timeline.length > 0 ? blocksFromTimeline(timeline) : blocksFromStatus(status);

  return (
    <div className="uptime-strip-block" aria-label={title}>
      <div className="uptime-strip-header">
        <span>{title}</span>
        <span className="uptime-strip-legend">
          <span>
            <i className="block up" /> Up
          </span>
          <span>
            <i className="block down" /> Down
          </span>
          <span>
            <i className="block unknown" /> Unknown
          </span>
        </span>
      </div>
      <div className="uptime-strip" role="img" aria-label={`${title} visualization`}>
        {blocks.map((block, index) => (
          <span key={`${block}-${index}`} className={`uptime-block ${block}`} title={block} />
        ))}
      </div>
    </div>
  );
}
