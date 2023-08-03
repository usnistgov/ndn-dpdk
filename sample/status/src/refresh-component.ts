import { Component } from "preact";

export abstract class AbortableComponent<P = {}, S = {}> extends Component<P, S> {
  constructor() {
    super();
    this.signal = this.abort.signal;
  }

  override componentWillUnmount() {
    this.abort.abort();
  }

  protected readonly abort = new AbortController();
  protected readonly signal: AbortSignal;
}

export abstract class TimerRefreshComponent<P = {}, S = {}> extends Component<P, S> {
  override componentDidMount() {
    void this.processRefresh();
  }

  override componentWillUnmount() {
    clearTimeout(this.timer);
  }

  protected interval = 5000;

  protected abstract refresh(): Promise<Partial<S> | undefined>;

  private timer?: number;

  private readonly processRefresh = async () => {
    let update: Partial<S> | undefined;
    try {
      update = await this.refresh();
    } catch (err: unknown) {
      this.componentDidCatch?.(err, {});
    } finally {
      this.timer = setTimeout(this.processRefresh, this.interval * (1 + 0.1 * Math.random())) as unknown as number;
    }

    if (update) {
      this.setState(update);
    }
  };
}
