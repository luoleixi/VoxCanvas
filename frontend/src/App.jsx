import React from 'react';
import { createRoot } from 'react-dom/client';
import './styles.css';

function App() {
  return (
    <main className="app-shell">
      <section className="demo-panel" aria-label="VoxCanvas frontend demo">
        <p className="eyebrow">CI/CD Demo</p>
        <h1>VoxCanvas Frontend</h1>
        <p className="summary">
          This static React placeholder confirms the frontend build pipeline is wired correctly.
        </p>
      </section>
    </main>
  );
}

createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
