import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import StatusPage from "./StatusPage";
import "./styles.css";

const statusMatch = window.location.pathname.match(/^\/status\/([^/]+)\/?$/);
const slug = statusMatch ? decodeURIComponent(statusMatch[1]) : null;

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>{slug ? <StatusPage slug={slug} /> : <App />}</React.StrictMode>
);
