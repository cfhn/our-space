export const rfidToBase64 = (hex: string) =>
  btoa(hex.match(/\w{2}/g)?.map((a) => String.fromCharCode(parseInt(a, 16))).join("") ?? "")

export const base64ToRfid = (base64: string) => [...atob(base64)].map(c=> c.charCodeAt(0).toString(16).padStart(2,"0")).join('')
