# Frontend Style Skeleton

Use this as a copy-paste starting point when aligning a new Svelte surface with the current ContextOS visual pattern.

```css
.surface {
  min-width: 0;
  background: transparent;
  color: #1c1b18;
}

.section-row {
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid #d7d2c8;
  padding: 12px 14px;
}

.underline-button {
  min-height: 34px;
  border: 0;
  border-bottom: 1px solid #bdb7a8;
  border-radius: 0;
  background-color: transparent;
  background-image: linear-gradient(90deg, #1c1b18 0 50%, transparent 50% 100%);
  background-position: 100% 0;
  background-size: 200% 100%;
  color: #1c1b18;
  font-weight: 700;
  padding: 0 12px;
  transition:
    background-position 0.18s ease,
    color 0.15s,
    border-color 0.15s,
    opacity 0.15s;
}

.underline-button:hover:not(:disabled) {
  border-bottom-color: #1c1b18;
  background-position: 0 0;
  color: #f8f6ef;
}

.underline-button:disabled {
  cursor: not-allowed;
  opacity: 0.42;
}

.local-scroll {
  overflow: auto;
  overscroll-behavior: contain;
  scrollbar-width: none;
}

.local-scroll::-webkit-scrollbar {
  display: none;
}
```
