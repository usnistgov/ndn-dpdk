export as namespace ethface;

export interface Locator {
  Scheme: "ether";
  Port: string;
  Local: string;
  Remote: string;
}
