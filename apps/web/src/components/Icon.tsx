const paths = {
  home: "M3 11 12 3l9 8M5 10v11h5v-7h4v7h5V10",
  folder: "M3 7V4h7l3 3h8v14H3V7Zm0 3h18",
  library: "M4 4h4v16H4V4Zm6 0h4v16h-4V4Zm6 2 4 1-3 13-4-1 3-13Z",
  decisions: "M4 4h16v12H8l-4 4V4Zm4 5 2.5 2.5L16 7",
  arrow: "M4 12h16m-6-6 6 6-6 6",
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
