import Hls from "hls.js";
import { useEffect, useRef } from "react";

interface VideoPlayerProps {
  src: string;
}

const VideoPlayer: React.FC<VideoPlayerProps> = ({ src }) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    let hls: Hls | null = null;

    if (videoRef.current) {
      const videoElement = videoRef.current;

      if (Hls.isSupported()) {
        console.log("HLS is here");

        hls = new Hls();

        hls.loadSource(src);
        hls.attachMedia(videoElement);
        hls.on(Hls.Events.MANIFEST_PARSED, () => {
          console.log("Manifest good");
          videoElement.play().catch((e) => {
            console.error("vidplayer fail", e);
          });
        });

        hls.on(Hls.Events.ERROR, (event, data) => {
          if (data.fatal) {
            console.error("HLS JS FATAL ERROR");
          } else {
            console.warn("NON FATAL ERROR");
          }
        });
      } else if (videoElement.canPlayType("application/vnd.apple.mpegurl")) {
        console.log("Broser supports HLS native");

        videoElement.src = src;
        videoElement.addEventListener("loadedmetadata", () => {
          console.log("Metadata loaded, attempting to play using native");
          videoElement
            .play()
            .catch((error) => console.error("Native autoplay broken"));
        });
      } else {
        console.error("HLS not supported");
      }
    }
  });

  return (
    <video
      ref={videoRef}
      controls
      style={{ width: "100%", maxWidth: "800px" }}
    />
  );
};

export default VideoPlayer;
