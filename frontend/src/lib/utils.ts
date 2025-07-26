import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatProtoOrDate(
  input?: { seconds: number; nanos: number } | Date
): string {
  if (!input) return "N/A";

  let date: Date;

  if (input instanceof Date) {
    date = input;
  } else if (
    typeof input.seconds === "number" &&
    typeof input.nanos === "number"
  ) {
    const millis = input.seconds * 1000 + Math.floor(input.nanos / 1e6);
    date = new Date(millis);
  } else {
    return "Invalid date";
  }

  return date.toLocaleString("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}