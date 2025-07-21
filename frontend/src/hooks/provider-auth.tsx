import { useFlag } from "@unleash/proxy-client-react";
import { BASE_PROVIDER_OPTIONS } from "@/models/provider-auth";

export function useProviderOptions(): string[] {
  const isNewProviderEnabled = useFlag("kind.cluster");

  return isNewProviderEnabled
    ? [...BASE_PROVIDER_OPTIONS, "KIND"]
    : BASE_PROVIDER_OPTIONS;
}
