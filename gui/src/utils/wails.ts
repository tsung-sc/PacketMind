type WailsRuntime = {
  ClipboardSetText: (text: string) => Promise<boolean>;
  WindowGetSize: () => Promise<{ w: number; h: number }>;
  EventsOn: (event: string, callback: (...data: any[]) => void) => () => void;
  EventsOff: (event: string, ...additionalEvents: string[]) => void;
};

declare global {
  interface Window {
    wails?: any;
    go?: {
      main?: {
        [key: string]: {
          [method: string]: (...args: any[]) => Promise<any>;
        };
      };
    };
  }
}

export const copyToClipboard = async (text: string): Promise<boolean> => {
  try {
    if (window.wails === undefined) {
      await navigator.clipboard.writeText(text);
      return true;
    }
    
    const { ClipboardSetText } = await import('../../wailsjs/runtime/runtime');
    return await ClipboardSetText(text);
  } catch (error) {
    console.error('Failed to copy text:', error);
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch (fallbackError) {
      console.error('Fallback copy failed:', fallbackError);
      return false;
    }
  }
};

export const isWailsRuntime = (): boolean => {
	return window.wails !== undefined;
};

export const getWindowSize = async (): Promise<{ width: number; height: number }> => {
  try {
    if (!isWailsRuntime()) {
      return {
        width: window.innerWidth,
        height: window.innerHeight
      };
    }
    
    const { WindowGetSize } = await import('../../wailsjs/runtime/runtime');
    const size = await WindowGetSize();
    return { width: size.w, height: size.h };
  } catch (error) {
    console.error('Failed to get window size from Wails:', error);
    return {
      width: window.innerWidth,
      height: window.innerHeight
    };
  }
};

export const onWindowResize = (callback: (data: { width: number; height: number }) => void) => {
  if (!isWailsRuntime()) {
    const handler = () => callback({ width: window.innerWidth, height: window.innerHeight });
    window.addEventListener('resize', handler);
    return () => window.removeEventListener('resize', handler);
  }
  
  (async () => {
    try {
      const { EventsOn, EventsOff } = await import('../../wailsjs/runtime/runtime');
      EventsOn('wails:window:resize', callback);
      return () => EventsOff('wails:window:resize');
    } catch (error) {
      console.error('Failed to bind Wails resize event:', error);
      const handler = () => callback({ width: window.innerWidth, height: window.innerHeight });
      window.addEventListener('resize', handler);
      return () => window.removeEventListener('resize', handler);
    }
  })();
  
  return () => {};
};
