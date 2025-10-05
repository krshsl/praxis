// TypeScript declaration for use-sound (temporary workaround)
declare module 'use-sound' {
  type Options = {
    volume?: number;
    interrupt?: boolean;
    soundEnabled?: boolean;
    onend?: () => void;
    [key: string]: any;
  };
  type PlayOptions = {
    sound?: string;
    [key: string]: any;
  };
  type UseSoundReturn = [
    (options?: PlayOptions) => void,
    { stop: () => void }
  ];
  export default function useSound(
    url: string,
    options?: Options
  ): UseSoundReturn;
}
