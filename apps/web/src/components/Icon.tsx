const paths = {
  home: "M3 11 12 3l9 8M5 10v11h5v-7h4v7h5V10",
  folder: "M3 7V4h7l3 3h8v14H3V7Zm0 3h18",
  library: "M4 4h4v16H4V4Zm6 0h4v16h-4V4Zm6 2 4 1-3 13-4-1 3-13Z",
  decisions: "M4 4h16v12H8l-4 4V4Zm4 5 2.5 2.5L16 7",
  arrow: "M4 12h16m-6-6 6 6-6 6",
  key: "M15 7a4 4 0 1 1-3.9 4.9L4 19v2h3v-2h2v-2h2l1.9-1.9A4 4 0 0 1 15 7Zm1 2h.01",
  workflow: "M4 5h6v4H4V5Zm10 0h6v4h-6V5ZM9 15h6v4H9v-4Zm-2-6v2a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2V9",
  shield: "M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6l8-3Zm-3 9 2 2 4-4",
  sun: "M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Zm0-6v2m0 16v2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4",
  moon: "M20 14.5A8 8 0 0 1 9.5 4 8 8 0 1 0 20 14.5Z",
} as const;

export type IconName = keyof typeof paths;

export function Icon({ name }: { name: IconName }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={paths[name]} />
    </svg>
  );
}
