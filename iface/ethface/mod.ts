export interface Locator {
  Scheme: "ether";
  Port: string;
  Local?: string;
  Remote?: string;

  /**
   * @items.type integer
   * @items.minimum 1
   * @items.maximum 4095
   */
  Vlan?: number[];
}
