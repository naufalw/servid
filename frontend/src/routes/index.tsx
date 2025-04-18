import { createFileRoute } from "@tanstack/react-router";
import { FormEvent, useState } from "react";
import VideoPlayer from "~/components/VideoPlayer";

export const Route = createFileRoute("/")({
  component: Home,
});

function Home() {
  const [message, setMessage] = useState<string>("");
  const [isUploading, setIsUploading] = useState<boolean>(false);
  const [streamUrl, setStreamUrl] = useState<string | null>(
    "http://192.168.0.96:8080/stream/78f3a72c-f112-47c0-9308-9c7b3c699734/master.m3u8",
  );

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setStreamUrl(null);

    setIsUploading(true);

    setMessage("Uploading");

    const form = event.currentTarget;
    const fileInput = form.elements.namedItem("videoFile") as HTMLInputElement;

    if (!fileInput || !fileInput.files || fileInput.files.length == 0) {
      setMessage("Select file");
      setIsUploading(false);
      return;
    }

    const file = fileInput.files[0];
    const formData = new FormData();

    formData.append("video", file);

    const backendUrl = "http://192.168.0.96:8080/upload";

    try {
      const response = await fetch(backendUrl, {
        method: "POST",
        body: formData,
      });

      const result = await response.text();

      if (!response.ok) throw new Error("Error" + response.status + result);
      const urlMatch = result.match(/Stream URL: (\/stream\/.*\.m3u8)/);
      const relativeUrl = urlMatch![1];

      const fullStreamUrl = `http://192.168.0.96:8080${relativeUrl}`;
      setStreamUrl(fullStreamUrl);

      setMessage("upload good");
    } catch (e) {
      console.error(e);
      setMessage(
        "upload error " + (e instanceof Error ? e.message : String(e)),
      );
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="p-2">
      <h1>Video upload</h1>
      <form onSubmit={handleSubmit}>
        <input type="file" name="videoFile" accept="video/*" required />
        <button type="submit" disabled={isUploading}>
          {isUploading ? "uploading" : "upload"}
        </button>
        {message && <p>{message}</p>}

        {streamUrl && (
          <div>
            <h2>Playback</h2>
            <VideoPlayer src={streamUrl} />
          </div>
        )}
      </form>
    </div>
  );
}
