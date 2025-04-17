import { createFileRoute } from "@tanstack/react-router";
import { FormEvent, useState } from "react";

export const Route = createFileRoute("/")({
  component: Home,
});

function Home() {
  const [message, setMessage] = useState<string>("");
  const [isUploading, setIsUploading] = useState<boolean>(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

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

    const backendUrl = "http://127.0.0.1:8080/upload";

    try {
      const response = await fetch(backendUrl, {
        method: "POST",
        body: formData,
      });

      const result = await response.text();

      if (!response.ok) throw new Error("Error" + response.status + result);

      setMessage("upload good");
    } catch (e) {
      console.error(e);
      setMessage(
        "upload error " + (e instanceof Error ? e.message : String(e)),
      );
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
      </form>
    </div>
  );
}
