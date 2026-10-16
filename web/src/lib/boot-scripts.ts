// Inline scripts that run in <head> before first paint. Kept out of client
// modules so the server layout receives plain strings.

const THEME_KEY = "yellow:theme";

/** Applies the saved or system theme so there is no flash of the wrong one. */
export const themeScript = `(function(){try{var p=localStorage.getItem("${THEME_KEY}")||"system";var d=p==="dark"||(p==="system"&&matchMedia("(prefers-color-scheme: dark)").matches);document.documentElement.classList.toggle("dark",d);}catch(e){}})();`;

/**
 * If this browser was connected last time, marks <html> so CSS shows a loading
 * state instead of flashing the connect screen while the wallet reconnects.
 */
export const walletHintScript = `(function(){try{if(localStorage.getItem("wagmi.recentConnectorId")||localStorage.getItem("@appkit/connection_status")==='"connected"')document.documentElement.dataset.wallet="1";}catch(e){}})();`;
