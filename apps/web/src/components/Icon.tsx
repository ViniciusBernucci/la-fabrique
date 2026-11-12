const paths = {
  home: "M3 11 12 3l9 8M5 10v11h5v-7h4v7h5V10",
  folder: "M3 7V4h7l3 3h8v14H3V7Zm0 3h18",
  library: "M4 4h4v16H4V4Zm6 0h4v16h-4V4Zm6 2 4 1-3 13-4-1 3-13Z",
  decisions: "M4 4h16v12H8l-4 4V4Zm4 5 2.5 2.5L16 7",
  arrow: "M4 12h16m-6-6 6 6-6 6",
  key: "M15 7a4 4 0 1 1-3.9 4.9L4 19v2h3v-2h2v-2h2l1.9-1.9A4 4 0 0 1 15 7Zm1 2h.01",
  workflow: "M4 5h6v4H4V5Zm10 0h6v4h-6V5ZM9 15h6v4H9v-4Zm-2-6v2a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2V9",
  shield: "M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6l8-3Zm-3 9 2 2 4-4",
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
