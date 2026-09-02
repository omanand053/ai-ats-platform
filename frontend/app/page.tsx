"use client";

import { useEffect, useState } from "react";

export default function Home() {
  const [data, setData] = useState<any>(null);

  useEffect(() => {
    fetch(`${process.env.NEXT_PUBLIC_API_URL}/health`)
      .then((res) => res.json())
      .then((json) => setData(json))
      .catch((err) => console.error(err));
  }, []);

  return (
    <main style={{ padding: "40px", fontFamily: "Arial" }}>
      <h1>AI ATS Platform</h1>

      {data ? (
        <>
          <p><strong>Status:</strong> {data.status}</p>
          <p><strong>Message:</strong> {data.message}</p>
        </>
      ) : (
        <p>Loading...</p>
      )}
    </main>
  );
}