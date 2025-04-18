"use client";
import React, { useEffect, useRef, useState, useCallback } from "react";
import Hls, { Level } from "hls.js";

interface VideoPlayerProps {
  src: string; // master.m3u8 URL
}

export default function VideoPlayerEnhanced({ src }: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [hls, setHls] = useState<Hls | null>(null);
  const [levels, setLevels] = useState<Level[]>([]);
  const [currentLevel, setCurrentLevel] = useState<number>(-1); // -1 = auto
  const [bandwidth, setBandwidth] = useState<number>(0);
  const [bufferLen, setBufferLen] = useState<number>(0);

  // Initialize HLS.js
  useEffect(() => {
    if (!videoRef.current) return;

    let h = new Hls({});
    h.attachMedia(videoRef.current);
    h.on(Hls.Events.MEDIA_ATTACHED, () => {
      h.loadSource(src);
    });

    h.on(Hls.Events.MANIFEST_PARSED, () => {
      setLevels(h.levels);
      setCurrentLevel(h.currentLevel);
    });

    h.on(Hls.Events.LEVEL_SWITCHED, (_, data) => {
      setCurrentLevel(data.level);
    });

    setHls(h);

    return () => {
      h.destroy();
      setHls(null);
    };
  }, [src]);

  // Poll stats every second
  useEffect(() => {
    if (!hls || !videoRef.current) return;
    const vid = videoRef.current;
    const interval = setInterval(() => {
      // bandwidthEstimate is in bits/sec
      setBandwidth(Math.round(hls.bandwidthEstimate / 1000)); // kb/s
      // buffer length = end of buffer range minus currentTime
      const buf = vid.buffered;
      if (buf.length) {
        setBufferLen(Math.round(buf.end(buf.length - 1) - vid.currentTime));
      }
    }, 1000);
    return () => clearInterval(interval);
  }, [hls]);

  // Handle quality change
  const onSelectChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      const lvl = parseInt(e.currentTarget.value);
      if (!hls) return;
      // -1 = AUTO
      hls.currentLevel = lvl;
      setCurrentLevel(lvl);
    },
    [hls],
  );

  return (
    <div style={{ maxWidth: 800 }}>
      <video
        ref={videoRef}
        controls
        style={{ width: "100%", background: "#000" }}
      />

      <div style={{ marginTop: 8 }}>
        <label>
          Quality:{" "}
          <select
            value={currentLevel}
            onChange={onSelectChange}
            disabled={!levels.length}
          >
            <option value={-1}>Auto</option>
            {levels.map((l, idx) => {
              const w = l.width;
              const h = l.height;
              const kb = Math.round(l.bitrate / 1000);
              return (
                <option value={idx} key={idx}>
                  {w}×{h} @ {kb}kbps
                </option>
              );
            })}
          </select>
        </label>
      </div>
      {/* Stats */}
      <div style={{ marginTop: 8, fontSize: 14, color: "#444" }}>
        <span>Bandwidth: {bandwidth} kb/s</span> {" | "}
        <span>Buffer: {bufferLen} s</span> {" | "}
        <span>
          Level:{" "}
          {currentLevel < 0
            ? "Auto"
            : `${levels[currentLevel].width}×${levels[currentLevel].height}`}
        </span>
      </div>
    </div>
  );
}
